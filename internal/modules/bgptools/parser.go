package bgptools

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
)

type tableLine struct {
	CIDR     string          `json:"CIDR"`
	Prefix   string          `json:"prefix"`
	ASN      json.RawMessage `json:"ASN"`
	ASNLower json.RawMessage `json:"asn"`
	AS       json.RawMessage `json:"AS"`
}

func parseTableFile(path string) (*trieNode, *trieNode, int, map[int][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, nil, fmt.Errorf("failed to open table dataset: %w", err)
	}
	defer file.Close()

	ipv4Trie := newTrie()
	ipv6Trie := newTrie()
	prefixCount := 0
	asnPrefixes := make(map[int][]string)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var row tableLine
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}

		cidr := strings.TrimSpace(row.CIDR)
		if cidr == "" {
			cidr = strings.TrimSpace(row.Prefix)
		}
		if cidr == "" {
			continue
		}

		asn, ok := firstASN(row.ASN, row.ASNLower, row.AS)
		if !ok {
			continue
		}

		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		prefix = prefix.Masked()

		record := prefixRecord{
			CIDR: prefix.String(),
			ASN:  asn,
		}

		if prefix.Addr().Is4() {
			insertPrefix(ipv4Trie, prefix, record)
		} else {
			insertPrefix(ipv6Trie, prefix, record)
		}
		asnPrefixes[asn] = append(asnPrefixes[asn], record.CIDR)

		prefixCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, 0, nil, fmt.Errorf("failed to read table dataset: %w", err)
	}

	normalizePrefixLists(asnPrefixes)

	return ipv4Trie, ipv6Trie, prefixCount, asnPrefixes, nil
}

func normalizePrefixLists(values map[int][]string) {
	for asn, prefixes := range values {
		if len(prefixes) <= 1 {
			continue
		}

		slices.Sort(prefixes)
		deduped := prefixes[:1]
		for i := 1; i < len(prefixes); i++ {
			if prefixes[i] == prefixes[i-1] {
				continue
			}
			deduped = append(deduped, prefixes[i])
		}

		values[asn] = deduped
	}
}

func parseASNsFile(path string) (map[int]asnInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ASNs dataset: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	asns := make(map[int]asnInfo)
	rowIndex := 0

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to parse ASNs dataset: %w", err)
		}

		rowIndex++
		if rowIndex == 1 && len(record) > 0 && strings.Contains(strings.ToLower(record[0]), "asn") {
			continue
		}
		if len(record) < 4 {
			continue
		}

		asn, ok := parseASNString(record[0])
		if !ok {
			continue
		}

		asns[asn] = asnInfo{
			Name:    strings.TrimSpace(record[1]),
			Tag:     strings.TrimSpace(record[2]),
			Country: strings.TrimSpace(record[3]),
		}
	}

	return asns, nil
}

func firstASN(candidates ...json.RawMessage) (int, bool) {
	for _, candidate := range candidates {
		asn, ok := parseASNRaw(candidate)
		if ok {
			return asn, true
		}
	}
	return 0, false
}

func parseASNRaw(raw json.RawMessage) (int, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return 0, false
	}

	trimmed = strings.Trim(trimmed, "\"")
	return parseASNString(trimmed)
}

func parseASNString(raw string) (int, bool) {
	value := strings.TrimSpace(strings.ToUpper(raw))
	if value == "" {
		return 0, false
	}

	value = strings.TrimPrefix(value, "AS")
	asn, err := strconv.Atoi(value)
	if err != nil || asn <= 0 {
		return 0, false
	}

	return asn, true
}
