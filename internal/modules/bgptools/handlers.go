package bgptools

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type handler struct {
	svc *service
}

func newHandler(svc *service) *handler {
	return &handler{svc: svc}
}

func (h *handler) getHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Health())
}

func (h *handler) getIPLookup(w http.ResponseWriter, r *http.Request) {
	ips := parseIPInputs(r)
	if len(ips) == 0 {
		writeError(w, http.StatusBadRequest, `missing query parameter "ip"`)
		return
	}
	if len(ips) > h.svc.maxBatch() {
		writeError(w, http.StatusBadRequest, "too many IPs in one request")
		return
	}

	results, err := h.svc.LookupIPs(r.Context(), ips)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, batchIPResponse{
		Count:   len(results),
		Results: results,
	})
}

func (h *handler) postIPLookup(w http.ResponseWriter, r *http.Request) {
	var request ipBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(request.IPs) == 0 {
		writeError(w, http.StatusBadRequest, `missing field "ips"`)
		return
	}
	if len(request.IPs) > h.svc.maxBatch() {
		writeError(w, http.StatusBadRequest, "too many IPs in one request")
		return
	}

	results, err := h.svc.LookupIPs(r.Context(), request.IPs)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, batchIPResponse{
		Count:   len(results),
		Results: results,
	})
}

func (h *handler) getASNLookup(w http.ResponseWriter, r *http.Request) {
	asns, err := parseASNInputs(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(asns) == 0 {
		writeError(w, http.StatusBadRequest, `missing query parameter "asn"`)
		return
	}
	if len(asns) > h.svc.maxBatch() {
		writeError(w, http.StatusBadRequest, "too many ASNs in one request")
		return
	}

	results, lookupErr := h.svc.LookupASNs(r.Context(), asns)
	if lookupErr != nil {
		writeError(w, http.StatusServiceUnavailable, lookupErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, batchASNResponse{
		Count:   len(results),
		Results: results,
	})
}

func (h *handler) getASNByPath(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "asn")
	asn, ok := parseSingleASN(raw)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid ASN in path")
		return
	}

	results, err := h.svc.LookupASNs(r.Context(), []int{asn})
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results[0])
}

func (h *handler) postASNLookup(w http.ResponseWriter, r *http.Request) {
	var request asnBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(request.ASNs) == 0 {
		writeError(w, http.StatusBadRequest, `missing field "asns"`)
		return
	}
	if len(request.ASNs) > h.svc.maxBatch() {
		writeError(w, http.StatusBadRequest, "too many ASNs in one request")
		return
	}

	results, err := h.svc.LookupASNs(r.Context(), request.ASNs)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, batchASNResponse{
		Count:   len(results),
		Results: results,
	})
}

func (h *handler) postReload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.svc.cfg.HTTPTimeout)
	defer cancel()

	if err := h.svc.reload(ctx, true); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.svc.Health())
}

func parseIPInputs(r *http.Request) []string {
	return parseIPValues(r.URL.Query()["ip"])
}

func parseASNInputs(r *http.Request) ([]int, error) {
	return parseASNValues(r.URL.Query()["asn"])
}

func parseIPValues(raw []string) []string {
	out := make([]string, 0)
	for _, item := range raw {
		for _, token := range strings.Split(item, ",") {
			value := strings.TrimSpace(token)
			if value == "" {
				continue
			}
			out = append(out, value)
		}
	}
	return out
}

func parseASNValues(raw []string) ([]int, error) {
	out := make([]int, 0)

	for _, item := range raw {
		for _, token := range strings.Split(item, ",") {
			value := strings.TrimSpace(token)
			if value == "" {
				continue
			}

			asn, ok := parseSingleASN(value)
			if !ok {
				return nil, &parseError{Message: "invalid ASN in query"}
			}
			out = append(out, asn)
		}
	}

	return out, nil
}

func parseSingleASN(raw string) (int, bool) {
	value := strings.TrimSpace(strings.ToUpper(raw))
	value = strings.TrimPrefix(value, "AS")

	asn, err := strconv.Atoi(value)
	if err != nil || asn <= 0 {
		return 0, false
	}

	return asn, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

type parseError struct {
	Message string
}

func (e *parseError) Error() string {
	return e.Message
}
