package server

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

const (
	startupLabelLocal   = "Local URL"
	startupLabelNetwork = "Network URL"
	startupLabelPublic  = "Public URL"
)

// StartupURL is a suggested URL displayed at application startup.
type StartupURL struct {
	Label string
	URL   string
}

// DiscoverStartupURLs returns human-friendly URLs for the current bind address.
func DiscoverStartupURLs(listenAddr string) []StartupURL {
	return startupURLsFromAddrs(listenAddr, interfaceAddresses())
}

func startupURLsFromAddrs(listenAddr string, addrs []netip.Addr) []StartupURL {
	host, port := splitListenAddress(listenAddr)
	if port == "" {
		return nil
	}

	switch {
	case isWildcardHost(host):
		return wildcardURLs(port, addrs)
	default:
		return []StartupURL{{
			Label: classifyHost(host),
			URL:   buildURL(host, port),
		}}
	}
}

func wildcardURLs(port string, addrs []netip.Addr) []StartupURL {
	urls := []StartupURL{{
		Label: startupLabelLocal,
		URL:   buildURL("localhost", port),
	}}

	var privateIP netip.Addr
	var publicIP netip.Addr

	for _, addr := range addrs {
		normalized := addr.Unmap()
		if !normalized.IsValid() || normalized.IsLoopback() {
			continue
		}

		if publicIP.IsValid() && privateIP.IsValid() {
			break
		}

		if isPublicAddress(normalized) {
			if !publicIP.IsValid() {
				publicIP = normalized
			}
			continue
		}

		if isPrivateAddress(normalized) && !privateIP.IsValid() {
			privateIP = normalized
		}
	}

	if publicIP.IsValid() {
		urls = append(urls, StartupURL{
			Label: startupLabelPublic,
			URL:   buildURL(publicIP.String(), port),
		})
		return dedupeURLs(urls)
	}

	if privateIP.IsValid() {
		urls = append(urls, StartupURL{
			Label: startupLabelNetwork,
			URL:   buildURL(privateIP.String(), port),
		})
	}

	return dedupeURLs(urls)
}

func splitListenAddress(listenAddr string) (string, string) {
	if strings.HasPrefix(listenAddr, ":") {
		return "", strings.TrimPrefix(listenAddr, ":")
	}

	host, port, err := net.SplitHostPort(listenAddr)
	if err == nil {
		return strings.Trim(host, "[]"), port
	}

	return "", ""
}

func classifyHost(host string) string {
	normalized := strings.Trim(host, "[]")
	if normalized == "" {
		return startupLabelLocal
	}

	if strings.EqualFold(normalized, "localhost") {
		return startupLabelLocal
	}

	if ip, err := netip.ParseAddr(normalized); err == nil {
		addr := ip.Unmap()
		switch {
		case addr.IsLoopback():
			return startupLabelLocal
		case isPublicAddress(addr):
			return startupLabelPublic
		default:
			return startupLabelNetwork
		}
	}

	return startupLabelNetwork
}

func buildURL(host, port string) string {
	trimmedHost := strings.Trim(host, "[]")
	if ip, err := netip.ParseAddr(trimmedHost); err == nil && ip.Is6() {
		return fmt.Sprintf("http://[%s]:%s", ip.String(), port)
	}

	return fmt.Sprintf("http://%s:%s", trimmedHost, port)
}

func isWildcardHost(host string) bool {
	normalized := strings.Trim(strings.TrimSpace(host), "[]")
	return normalized == "" || normalized == "0.0.0.0" || normalized == "::"
}

func isPublicAddress(addr netip.Addr) bool {
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !addr.IsLoopback()
}

func isPrivateAddress(addr netip.Addr) bool {
	return addr.IsPrivate() || addr.IsLinkLocalUnicast()
}

func interfaceAddresses() []netip.Addr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var results []netip.Addr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, raw := range addrs {
			ip := extractIP(raw)
			if ip == nil {
				continue
			}

			addr, ok := netip.AddrFromSlice(ip)
			if !ok {
				continue
			}

			results = append(results, addr.Unmap())
		}
	}

	return results
}

func extractIP(raw net.Addr) net.IP {
	switch value := raw.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		return nil
	}
}

func dedupeURLs(urls []StartupURL) []StartupURL {
	seen := make(map[string]struct{}, len(urls))
	result := make([]StartupURL, 0, len(urls))

	for _, item := range urls {
		if _, exists := seen[item.URL]; exists {
			continue
		}
		seen[item.URL] = struct{}{}
		result = append(result, item)
	}

	return result
}
