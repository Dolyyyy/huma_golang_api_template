package bgptools

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type getIPLookupInput struct {
	IP []string `query:"ip" doc:"One or many IPs. Supports repeated values and comma-separated values."`
}

type getASNLookupInput struct {
	ASN []string `query:"asn" doc:"One or many ASNs. Supports repeated values and comma-separated values."`
}

type listASNsInput struct {
	Q              string `query:"q" doc:"Optional search filter across ASN number, name, tag and country."`
	Tag            string `query:"tag" doc:"Optional exact class filter (Unknown, Eyeball, Content, Carrier, T1)."`
	ExcludeUnknown bool   `query:"exclude_unknown" doc:"When true, exclude entries where tag/class is Unknown."`
	Offset         int    `query:"offset" minimum:"0" doc:"Start index in the ASN list (default: 0)."`
	Limit          int    `query:"limit" minimum:"0" doc:"Max number of items to return. 0 means all."`
}

type getASNByPathInput struct {
	ASN string `path:"asn" doc:"ASN number, with or without the AS prefix (example: 13335 or AS13335)."`
}

type listASNPrefixesInput struct {
	ASN    string `path:"asn" doc:"ASN number, with or without the AS prefix (example: 13335 or AS13335)."`
	Offset int    `query:"offset" minimum:"0" doc:"Start index in the prefix list (default: 0)."`
	Limit  int    `query:"limit" minimum:"0" doc:"Max number of prefixes to return. 0 means all."`
}

type postIPLookupInput struct {
	Body ipBatchRequest
}

type postASNLookupInput struct {
	Body asnBatchRequest
}

type healthOutput struct {
	Body healthResponse
}

type batchIPOutput struct {
	Body batchIPResponse
}

type batchASNOutput struct {
	Body batchASNResponse
}

type asnLookupOutput struct {
	Body asnLookupResult
}

type listASNsOutput struct {
	Body listASNsResponse
}

type listASNPrefixesOutput struct {
	Body listASNPrefixesResponse
}

func registerHumaRoutes(api huma.API) {
	if globalSvc == nil {
		return
	}

	handler := newHandler(globalSvc)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-health",
		Method:      http.MethodGet,
		Path:        "/bgptools/health",
		Summary:     "Get bgptools cache and dataset status",
		Tags:        []string{"bgptools"},
	}, handler.humaGetHealth)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-ip-lookup-get",
		Method:      http.MethodGet,
		Path:        "/bgptools/ip",
		Summary:     "Lookup one or many IP addresses",
		Tags:        []string{"bgptools"},
	}, handler.humaGetIPLookup)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-ip-lookup-post",
		Method:      http.MethodPost,
		Path:        "/bgptools/ip",
		Summary:     "Lookup one or many IP addresses from JSON body",
		Tags:        []string{"bgptools"},
	}, handler.humaPostIPLookup)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-asn-lookup-get",
		Method:      http.MethodGet,
		Path:        "/bgptools/asn",
		Summary:     "Lookup one or many ASNs",
		Tags:        []string{"bgptools"},
	}, handler.humaGetASNLookup)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-asn-list-all",
		Method:      http.MethodGet,
		Path:        "/bgptools/asns",
		Summary:     "List all ASNs from cached dataset",
		Tags:        []string{"bgptools"},
	}, handler.humaListASNs)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-asn-lookup-by-path",
		Method:      http.MethodGet,
		Path:        "/bgptools/asn/{asn}",
		Summary:     "Lookup one ASN by path parameter",
		Tags:        []string{"bgptools"},
	}, handler.humaGetASNByPath)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-asn-prefixes",
		Method:      http.MethodGet,
		Path:        "/bgptools/asn/{asn}/prefixes",
		Summary:     "List prefixes announced by one ASN",
		Tags:        []string{"bgptools"},
	}, handler.humaListASNPrefixes)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-asn-lookup-post",
		Method:      http.MethodPost,
		Path:        "/bgptools/asn",
		Summary:     "Lookup one or many ASNs from JSON body",
		Tags:        []string{"bgptools"},
	}, handler.humaPostASNLookup)

	huma.Register(api, huma.Operation{
		OperationID: "bgptools-reload",
		Method:      http.MethodPost,
		Path:        "/bgptools/reload",
		Summary:     "Force dataset reload",
		Tags:        []string{"bgptools"},
	}, handler.humaPostReload)
}

func (h *handler) humaGetHealth(context.Context, *struct{}) (*healthOutput, error) {
	return &healthOutput{Body: h.svc.Health()}, nil
}

func (h *handler) humaGetIPLookup(ctx context.Context, input *getIPLookupInput) (*batchIPOutput, error) {
	ips := parseIPValues(input.IP)
	if len(ips) == 0 {
		return nil, huma.Error400BadRequest(`missing query parameter "ip"`)
	}
	if len(ips) > h.svc.maxBatch() {
		return nil, huma.Error400BadRequest("too many IPs in one request")
	}

	results, err := h.svc.LookupIPs(ctx, ips)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &batchIPOutput{
		Body: batchIPResponse{
			Count:   len(results),
			Results: results,
		},
	}, nil
}

func (h *handler) humaPostIPLookup(ctx context.Context, input *postIPLookupInput) (*batchIPOutput, error) {
	if len(input.Body.IPs) == 0 {
		return nil, huma.Error400BadRequest(`missing field "ips"`)
	}
	if len(input.Body.IPs) > h.svc.maxBatch() {
		return nil, huma.Error400BadRequest("too many IPs in one request")
	}

	results, err := h.svc.LookupIPs(ctx, input.Body.IPs)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &batchIPOutput{
		Body: batchIPResponse{
			Count:   len(results),
			Results: results,
		},
	}, nil
}

func (h *handler) humaGetASNLookup(ctx context.Context, input *getASNLookupInput) (*batchASNOutput, error) {
	asns, err := parseASNValues(input.ASN)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	if len(asns) == 0 {
		return nil, huma.Error400BadRequest(`missing query parameter "asn"`)
	}
	if len(asns) > h.svc.maxBatch() {
		return nil, huma.Error400BadRequest("too many ASNs in one request")
	}

	results, lookupErr := h.svc.LookupASNs(ctx, asns)
	if lookupErr != nil {
		return nil, huma.Error503ServiceUnavailable(lookupErr.Error())
	}

	return &batchASNOutput{
		Body: batchASNResponse{
			Count:   len(results),
			Results: results,
		},
	}, nil
}

func (h *handler) humaGetASNByPath(ctx context.Context, input *getASNByPathInput) (*asnLookupOutput, error) {
	asn, ok := parseSingleASN(input.ASN)
	if !ok {
		return nil, huma.Error400BadRequest("invalid ASN in path")
	}

	results, err := h.svc.LookupASNs(ctx, []int{asn})
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &asnLookupOutput{Body: results[0]}, nil
}

func (h *handler) humaListASNPrefixes(ctx context.Context, input *listASNPrefixesInput) (*listASNPrefixesOutput, error) {
	asn, ok := parseSingleASN(input.ASN)
	if !ok {
		return nil, huma.Error400BadRequest("invalid ASN in path")
	}
	if input.Offset < 0 {
		return nil, huma.Error400BadRequest("offset must be >= 0")
	}
	if input.Limit < 0 {
		return nil, huma.Error400BadRequest("limit must be >= 0")
	}

	response, err := h.svc.ListASNPrefixes(ctx, asn, input.Offset, input.Limit)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &listASNPrefixesOutput{Body: response}, nil
}

func (h *handler) humaListASNs(ctx context.Context, input *listASNsInput) (*listASNsOutput, error) {
	if input.Offset < 0 {
		return nil, huma.Error400BadRequest("offset must be >= 0")
	}
	if input.Limit < 0 {
		return nil, huma.Error400BadRequest("limit must be >= 0")
	}

	response, err := h.svc.ListASNs(ctx, input.Q, input.Tag, input.ExcludeUnknown, input.Offset, input.Limit)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &listASNsOutput{Body: response}, nil
}

func (h *handler) humaPostASNLookup(ctx context.Context, input *postASNLookupInput) (*batchASNOutput, error) {
	if len(input.Body.ASNs) == 0 {
		return nil, huma.Error400BadRequest(`missing field "asns"`)
	}
	if len(input.Body.ASNs) > h.svc.maxBatch() {
		return nil, huma.Error400BadRequest("too many ASNs in one request")
	}

	results, err := h.svc.LookupASNs(ctx, input.Body.ASNs)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &batchASNOutput{
		Body: batchASNResponse{
			Count:   len(results),
			Results: results,
		},
	}, nil
}

func (h *handler) humaPostReload(ctx context.Context, _ *struct{}) (*healthOutput, error) {
	reloadCtx, cancel := context.WithTimeout(ctx, h.svc.cfg.HTTPTimeout)
	defer cancel()

	if err := h.svc.reload(reloadCtx, true); err != nil {
		return nil, huma.Error503ServiceUnavailable(err.Error())
	}

	return &healthOutput{Body: h.svc.Health()}, nil
}
