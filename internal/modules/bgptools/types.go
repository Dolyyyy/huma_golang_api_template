package bgptools

import "time"

type prefixRecord struct {
	CIDR string
	ASN  int
}

type asnInfo struct {
	Name    string `json:"name,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Country string `json:"country,omitempty"`
}

type dataset struct {
	IPv4        *trieNode
	IPv6        *trieNode
	ASNs        map[int]asnInfo
	ASNList     []asnDirectoryItem
	ASNPrefixes map[int][]string
	PrefixCount int
	LoadedAt    time.Time
}

type ipLookupResult struct {
	Input   string `json:"input"`
	IP      string `json:"ip,omitempty"`
	Found   bool   `json:"found"`
	ASN     int    `json:"asn,omitempty"`
	Name    string `json:"name,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Country string `json:"country,omitempty"`
	CIDR    string `json:"cidr,omitempty"`
	Source  string `json:"source,omitempty"`
	Error   string `json:"error,omitempty"`
}

type asnLookupResult struct {
	ASN     int    `json:"asn"`
	Found   bool   `json:"found"`
	Name    string `json:"name,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Country string `json:"country,omitempty"`
}

type asnDirectoryItem struct {
	ASN       int    `json:"asn"`
	Name      string `json:"name,omitempty"`
	Tag       string `json:"tag,omitempty"`
	Country   string `json:"country,omitempty"`
	searchKey string
}

type batchIPResponse struct {
	Count   int              `json:"count"`
	Results []ipLookupResult `json:"results"`
}

type batchASNResponse struct {
	Count   int               `json:"count"`
	Results []asnLookupResult `json:"results"`
}

type listASNsResponse struct {
	Total          int                `json:"total"`
	Count          int                `json:"count"`
	Offset         int                `json:"offset"`
	Limit          int                `json:"limit"`
	HasMore        bool               `json:"has_more"`
	Query          string             `json:"query,omitempty"`
	Tag            string             `json:"tag,omitempty"`
	ExcludeUnknown bool               `json:"exclude_unknown,omitempty"`
	Results        []asnDirectoryItem `json:"results"`
}

type listASNPrefixesResponse struct {
	ASN     int      `json:"asn"`
	Found   bool     `json:"found"`
	Name    string   `json:"name,omitempty"`
	Tag     string   `json:"tag,omitempty"`
	Country string   `json:"country,omitempty"`
	Total   int      `json:"total"`
	Count   int      `json:"count"`
	Offset  int      `json:"offset"`
	Limit   int      `json:"limit"`
	HasMore bool     `json:"has_more"`
	Results []string `json:"results"`
}

type ipBatchRequest struct {
	IPs []string `json:"ips"`
}

type asnBatchRequest struct {
	ASNs []int `json:"asns"`
}

type healthResponse struct {
	Loaded      bool   `json:"loaded"`
	PrefixCount int    `json:"prefix_count"`
	ASNCount    int    `json:"asn_count"`
	CacheDir    string `json:"cache_dir"`
	TableFile   string `json:"table_file"`
	ASNsFile    string `json:"asns_file"`
	LastUpdated string `json:"last_updated,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}
