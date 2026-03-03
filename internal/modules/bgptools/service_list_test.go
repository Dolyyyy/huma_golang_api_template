package bgptools

import "testing"

func TestDatasetListASNsSupportsSearchQuery(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNList: buildASNDirectory(map[int]asnInfo{
			13335: {Name: "Cloudflare, Inc.", Tag: "CLOUDFLARE", Country: "US"},
			15169: {Name: "Google LLC", Tag: "GOOGLE", Country: "US"},
			32934: {Name: "Meta Platforms", Tag: "META", Country: "IE"},
		}),
	}

	result := d.listASNs("as15169", "", false, 0, 0)
	if result.Total != 1 {
		t.Fatalf("expected total=1 for query, got %d", result.Total)
	}
	if result.Count != 1 || len(result.Results) != 1 {
		t.Fatalf("expected one result, got count=%d len=%d", result.Count, len(result.Results))
	}
	if result.Results[0].ASN != 15169 {
		t.Fatalf("expected ASN 15169, got %d", result.Results[0].ASN)
	}
	if result.Query != "as15169" {
		t.Fatalf("expected query echo %q, got %q", "as15169", result.Query)
	}
}

func TestDatasetListASNsSearchPagination(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNList: buildASNDirectory(map[int]asnInfo{
			13335: {Name: "Cloudflare, Inc.", Tag: "CLOUDFLARE", Country: "US"},
			15169: {Name: "Google LLC", Tag: "GOOGLE", Country: "US"},
			32934: {Name: "Meta Platforms", Tag: "META", Country: "IE"},
		}),
	}

	result := d.listASNs("us", "", false, 1, 1)
	if result.Total != 2 {
		t.Fatalf("expected total=2 for query us, got %d", result.Total)
	}
	if result.Count != 1 || len(result.Results) != 1 {
		t.Fatalf("expected one paginated result, got count=%d len=%d", result.Count, len(result.Results))
	}
	if result.Results[0].ASN != 15169 {
		t.Fatalf("expected second US ASN to be 15169, got %d", result.Results[0].ASN)
	}
	if result.HasMore {
		t.Fatal("expected has_more=false on final page")
	}
}

func TestDatasetListASNsExcludeUnknown(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNList: buildASNDirectory(map[int]asnInfo{
			13335: {Name: "Cloudflare, Inc.", Tag: "Unknown", Country: "US"},
			15169: {Name: "Google LLC", Tag: "Eyeball", Country: "US"},
			32934: {Name: "Meta Platforms", Tag: "Content", Country: "IE"},
		}),
	}

	result := d.listASNs("", "", true, 0, 0)
	if result.Total != 2 {
		t.Fatalf("expected total=2 after excluding unknown, got %d", result.Total)
	}
	if result.ExcludeUnknown != true {
		t.Fatal("expected exclude_unknown=true in response")
	}
	for _, item := range result.Results {
		if item.Tag == "Unknown" {
			t.Fatal("unexpected Unknown tag after exclude_unknown filter")
		}
	}
}

func TestDatasetListASNsTagFilter(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNList: buildASNDirectory(map[int]asnInfo{
			13335: {Name: "Cloudflare, Inc.", Tag: "Unknown", Country: "US"},
			15169: {Name: "Google LLC", Tag: "Eyeball", Country: "US"},
			32934: {Name: "Meta Platforms", Tag: "Content", Country: "IE"},
		}),
	}

	result := d.listASNs("", "content", false, 0, 0)
	if result.Total != 1 {
		t.Fatalf("expected total=1 for content tag, got %d", result.Total)
	}
	if len(result.Results) != 1 || result.Results[0].ASN != 32934 {
		t.Fatalf("expected ASN 32934 for tag content, got %+v", result.Results)
	}
	if result.Tag != "content" {
		t.Fatalf("expected tag echo content, got %q", result.Tag)
	}
}

func TestDatasetListASNPrefixesPagination(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNs: map[int]asnInfo{
			15169: {Name: "Google LLC", Tag: "Content", Country: "US"},
		},
		ASNPrefixes: map[int][]string{
			15169: {"1.1.1.0/24", "2.2.2.0/24", "3.3.3.0/24"},
		},
	}

	result := d.listASNPrefixes(15169, 1, 1)
	if !result.Found {
		t.Fatal("expected found=true")
	}
	if result.Total != 3 || result.Count != 1 {
		t.Fatalf("expected total=3/count=1, got total=%d count=%d", result.Total, result.Count)
	}
	if len(result.Results) != 1 || result.Results[0] != "2.2.2.0/24" {
		t.Fatalf("unexpected paginated prefixes: %+v", result.Results)
	}
	if !result.HasMore {
		t.Fatal("expected has_more=true")
	}
	if result.Name != "Google LLC" {
		t.Fatalf("expected name metadata, got %q", result.Name)
	}
}

func TestDatasetListASNPrefixesUnknownASN(t *testing.T) {
	t.Parallel()

	d := dataset{
		ASNs:        map[int]asnInfo{},
		ASNPrefixes: map[int][]string{},
	}

	result := d.listASNPrefixes(64512, 0, 0)
	if result.Found {
		t.Fatal("expected found=false")
	}
	if result.Total != 0 || result.Count != 0 {
		t.Fatalf("expected empty result, got total=%d count=%d", result.Total, result.Count)
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected no prefixes, got %+v", result.Results)
	}
}
