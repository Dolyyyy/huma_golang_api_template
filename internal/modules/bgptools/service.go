package bgptools

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var additionalReservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
}

type service struct {
	cfg    moduleConfig
	client *http.Client

	mu             sync.RWMutex
	cond           *sync.Cond
	loading        bool
	snapshot       *dataset
	lastError      string
	updaterStarted bool
}

func newService(cfg moduleConfig) *service {
	svc := &service{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
	svc.cond = sync.NewCond(&svc.mu)
	return svc
}

func (s *service) startUpdater() {
	s.mu.Lock()
	if s.updaterStarted {
		s.mu.Unlock()
		return
	}
	s.updaterStarted = true
	interval := s.cfg.UpdateInterval
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.HTTPTimeout)
			_ = s.reload(ctx, false)
			cancel()
		}
	}()
}

func (s *service) ensureLoaded(ctx context.Context) error {
	s.mu.RLock()
	ready := s.snapshot != nil
	s.mu.RUnlock()
	if ready {
		return nil
	}

	return s.reload(ctx, true)
}

func (s *service) reload(ctx context.Context, force bool) error {
	s.mu.Lock()
	for s.loading {
		s.cond.Wait()
	}
	needsSnapshot := s.snapshot == nil
	s.loading = true
	s.mu.Unlock()

	snapshot, err := s.loadSnapshot(ctx, force || needsSnapshot)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.loading = false
	s.cond.Broadcast()

	if err != nil {
		s.lastError = err.Error()
		if s.snapshot != nil {
			return nil
		}
		return err
	}

	if snapshot != nil {
		s.snapshot = snapshot
	}
	s.lastError = ""
	return nil
}

func (s *service) loadSnapshot(ctx context.Context, force bool) (*dataset, error) {
	tableChanged, err := s.fetchIfNeeded(ctx, s.cfg.TableURL, s.cfg.tablePath(), s.cfg.TableMaxAge)
	if err != nil {
		return nil, err
	}

	asnsChanged, err := s.fetchIfNeeded(ctx, s.cfg.ASNsURL, s.cfg.asnsPath(), s.cfg.ASNsMaxAge)
	if err != nil {
		return nil, err
	}

	if !force && !tableChanged && !asnsChanged {
		return nil, nil
	}

	ipv4Trie, ipv6Trie, prefixCount, asnPrefixes, err := parseTableFile(s.cfg.tablePath())
	if err != nil {
		return nil, err
	}

	asnMap, err := parseASNsFile(s.cfg.asnsPath())
	if err != nil {
		return nil, err
	}

	asnList := buildASNDirectory(asnMap)

	return &dataset{
		IPv4:        ipv4Trie,
		IPv6:        ipv6Trie,
		ASNs:        asnMap,
		ASNList:     asnList,
		ASNPrefixes: asnPrefixes,
		PrefixCount: prefixCount,
		LoadedAt:    time.Now().UTC(),
	}, nil
}

func (s *service) LookupIPs(ctx context.Context, inputs []string) ([]ipLookupResult, error) {
	if err := s.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	snapshot := s.snapshot
	s.mu.RUnlock()
	if snapshot == nil {
		return nil, fmt.Errorf("bgptools dataset is not available")
	}

	results := make([]ipLookupResult, 0, len(inputs))
	for _, input := range inputs {
		results = append(results, snapshot.lookupIP(input))
	}

	return results, nil
}

func (s *service) LookupASNs(ctx context.Context, asns []int) ([]asnLookupResult, error) {
	if err := s.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	snapshot := s.snapshot
	s.mu.RUnlock()
	if snapshot == nil {
		return nil, fmt.Errorf("bgptools dataset is not available")
	}

	results := make([]asnLookupResult, 0, len(asns))
	for _, asn := range asns {
		results = append(results, snapshot.lookupASN(asn))
	}

	return results, nil
}

func (s *service) ListASNs(ctx context.Context, query, tag string, excludeUnknown bool, offset, limit int) (listASNsResponse, error) {
	if err := s.ensureLoaded(ctx); err != nil {
		return listASNsResponse{}, err
	}

	s.mu.RLock()
	snapshot := s.snapshot
	s.mu.RUnlock()
	if snapshot == nil {
		return listASNsResponse{}, fmt.Errorf("bgptools dataset is not available")
	}

	return snapshot.listASNs(query, tag, excludeUnknown, offset, limit), nil
}

func (s *service) ListASNPrefixes(ctx context.Context, asn, offset, limit int) (listASNPrefixesResponse, error) {
	if err := s.ensureLoaded(ctx); err != nil {
		return listASNPrefixesResponse{}, err
	}

	s.mu.RLock()
	snapshot := s.snapshot
	s.mu.RUnlock()
	if snapshot == nil {
		return listASNPrefixesResponse{}, fmt.Errorf("bgptools dataset is not available")
	}

	return snapshot.listASNPrefixes(asn, offset, limit), nil
}

func (s *service) Health() healthResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := healthResponse{
		Loaded:    s.snapshot != nil,
		CacheDir:  s.cfg.CacheDir,
		TableFile: filepath.ToSlash(s.cfg.tablePath()),
		ASNsFile:  filepath.ToSlash(s.cfg.asnsPath()),
		LastError: s.lastError,
	}

	if s.snapshot != nil {
		response.PrefixCount = s.snapshot.PrefixCount
		response.ASNCount = len(s.snapshot.ASNs)
		response.LastUpdated = s.snapshot.LoadedAt.Format(time.RFC3339)
	}

	return response
}

func (s *service) maxBatch() int {
	return s.cfg.MaxBatch
}

func buildASNDirectory(asnMap map[int]asnInfo) []asnDirectoryItem {
	keys := make([]int, 0, len(asnMap))
	for asn := range asnMap {
		keys = append(keys, asn)
	}
	sort.Ints(keys)

	out := make([]asnDirectoryItem, 0, len(keys))
	for _, asn := range keys {
		info := asnMap[asn]
		asnText := strconv.Itoa(asn)
		out = append(out, asnDirectoryItem{
			ASN:     asn,
			Name:    info.Name,
			Tag:     info.Tag,
			Country: info.Country,
			searchKey: strings.ToLower(
				asnText + " " +
					"as" + asnText + " " +
					info.Name + " " +
					info.Tag + " " +
					info.Country,
			),
		})
	}

	return out
}

func (d *dataset) listASNs(query, tag string, excludeUnknown bool, offset, limit int) listASNsResponse {
	trimmedQuery := strings.TrimSpace(query)
	normalizedTag := strings.ToLower(strings.TrimSpace(tag))
	source := d.ASNList
	if trimmedQuery != "" || normalizedTag != "" || excludeUnknown {
		needle := strings.ToLower(trimmedQuery)
		filtered := make([]asnDirectoryItem, 0, len(d.ASNList))
		for _, item := range d.ASNList {
			itemTag := strings.ToLower(strings.TrimSpace(item.Tag))
			if excludeUnknown && itemTag == "unknown" {
				continue
			}
			if normalizedTag != "" && itemTag != normalizedTag {
				continue
			}
			if trimmedQuery != "" && !strings.Contains(item.searchKey, needle) {
				continue
			}

			filtered = append(filtered, item)
		}
		source = filtered
	}

	total := len(source)

	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit < 0 {
		limit = 0
	}

	end := total
	if limit > 0 {
		end = offset + limit
		if end > total {
			end = total
		}
	}

	results := make([]asnDirectoryItem, end-offset)
	copy(results, source[offset:end])

	return listASNsResponse{
		Total:          total,
		Count:          len(results),
		Offset:         offset,
		Limit:          limit,
		HasMore:        end < total,
		Query:          trimmedQuery,
		Tag:            strings.TrimSpace(tag),
		ExcludeUnknown: excludeUnknown,
		Results:        results,
	}
}

func (d *dataset) listASNPrefixes(asn, offset, limit int) listASNPrefixesResponse {
	prefixes := d.ASNPrefixes[asn]
	total := len(prefixes)

	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit < 0 {
		limit = 0
	}

	end := total
	if limit > 0 {
		end = offset + limit
		if end > total {
			end = total
		}
	}

	results := make([]string, end-offset)
	copy(results, prefixes[offset:end])

	info, found := d.ASNs[asn]

	return listASNPrefixesResponse{
		ASN:     asn,
		Found:   found || total > 0,
		Name:    info.Name,
		Tag:     info.Tag,
		Country: info.Country,
		Total:   total,
		Count:   len(results),
		Offset:  offset,
		Limit:   limit,
		HasMore: end < total,
		Results: results,
	}
}

func (d *dataset) lookupIP(input string) ipLookupResult {
	result := ipLookupResult{
		Input: strings.TrimSpace(input),
		Found: false,
	}

	addr, normalized, err := parseInputIP(input)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.IP = normalized
	if isNonRoutable(addr) {
		result.Found = true
		result.ASN = 0
		result.Name = "Local/Private Network"
		result.Tag = "LOCAL"
		result.Country = "ZZ"
		result.Source = "private-range"
		return result
	}

	var match *prefixRecord
	if addr.Is4() {
		match = longestMatch(d.IPv4, addr)
	} else {
		match = longestMatch(d.IPv6, addr)
	}
	if match == nil {
		result.Source = "not-found"
		return result
	}

	result.Found = true
	result.ASN = match.ASN
	result.CIDR = match.CIDR
	result.Source = "prefix-match"

	info, ok := d.ASNs[match.ASN]
	if !ok {
		result.Name = "Unknown ASN"
		result.Country = "ZZ"
		return result
	}

	result.Name = info.Name
	result.Tag = info.Tag
	result.Country = info.Country
	return result
}

func (d *dataset) lookupASN(asn int) asnLookupResult {
	result := asnLookupResult{
		ASN:   asn,
		Found: false,
	}

	info, ok := d.ASNs[asn]
	if !ok {
		return result
	}

	result.Found = true
	result.Name = info.Name
	result.Tag = info.Tag
	result.Country = info.Country
	return result
}

func parseInputIP(raw string) (netip.Addr, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return netip.Addr{}, "", fmt.Errorf("empty IP")
	}

	if strings.Contains(trimmed, "/") {
		prefix, err := netip.ParsePrefix(trimmed)
		if err != nil {
			head, _, _ := strings.Cut(trimmed, "/")
			addr, parseErr := netip.ParseAddr(strings.TrimSpace(head))
			if parseErr != nil {
				return netip.Addr{}, "", fmt.Errorf("invalid IP %q", raw)
			}
			addr = addr.Unmap()
			return addr, addr.String(), nil
		}

		addr := prefix.Addr().Unmap()
		return addr, addr.String(), nil
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return netip.Addr{}, "", fmt.Errorf("invalid IP %q", raw)
	}
	addr = addr.Unmap()
	return addr, addr.String(), nil
}

func isNonRoutable(addr netip.Addr) bool {
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	if addr.IsLinkLocalMulticast() || addr.IsLinkLocalUnicast() {
		return true
	}

	for _, prefix := range additionalReservedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}
