package server

import (
	"net/netip"
	"testing"
)

func TestStartupURLsFromAddrs_WildcardWithPrivateAddress(t *testing.T) {
	t.Parallel()

	urls := startupURLsFromAddrs(":8888", []netip.Addr{
		netip.MustParseAddr("192.168.1.24"),
	})

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}

	if urls[0].Label != startupLabelLocal || urls[0].URL != "http://localhost:8888" {
		t.Fatalf("unexpected local URL: %+v", urls[0])
	}

	if urls[1].Label != startupLabelNetwork || urls[1].URL != "http://192.168.1.24:8888" {
		t.Fatalf("unexpected network URL: %+v", urls[1])
	}
}

func TestStartupURLsFromAddrs_WildcardPrefersPublicAddress(t *testing.T) {
	t.Parallel()

	urls := startupURLsFromAddrs(":8888", []netip.Addr{
		netip.MustParseAddr("192.168.1.24"),
		netip.MustParseAddr("8.8.8.8"),
	})

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}

	if urls[1].Label != startupLabelPublic || urls[1].URL != "http://8.8.8.8:8888" {
		t.Fatalf("expected public URL, got %+v", urls[1])
	}
}

func TestStartupURLsFromAddrs_WildcardPrefersPublicIPv4OverIPv6(t *testing.T) {
	t.Parallel()

	urls := startupURLsFromAddrs(":8888", []netip.Addr{
		netip.MustParseAddr("2a01:cb18:d44:d100:d777:3456:395d:8fc6"),
		netip.MustParseAddr("8.8.8.8"),
	})

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}

	if urls[1].Label != startupLabelPublic || urls[1].URL != "http://8.8.8.8:8888" {
		t.Fatalf("expected public IPv4 URL, got %+v", urls[1])
	}
}

func TestStartupURLsFromAddrs_ExplicitLocalhost(t *testing.T) {
	t.Parallel()

	urls := startupURLsFromAddrs("127.0.0.1:8888", nil)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	if urls[0].Label != startupLabelLocal || urls[0].URL != "http://127.0.0.1:8888" {
		t.Fatalf("unexpected explicit localhost URL: %+v", urls[0])
	}
}

func TestStartupURLsFromAddrs_ExplicitPublicAddress(t *testing.T) {
	t.Parallel()

	urls := startupURLsFromAddrs("34.120.0.10:8888", nil)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	if urls[0].Label != startupLabelPublic || urls[0].URL != "http://34.120.0.10:8888" {
		t.Fatalf("unexpected explicit public URL: %+v", urls[0])
	}
}
