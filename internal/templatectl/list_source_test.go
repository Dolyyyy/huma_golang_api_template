package templatectl

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseGitHubOwnerRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		owner  string
		repo   string
		ok     bool
	}{
		{
			name:   "https",
			source: "https://github.com/Dolyyyy/huma_golang_api_template_modules",
			owner:  "Dolyyyy",
			repo:   "huma_golang_api_template_modules",
			ok:     true,
		},
		{
			name:   "https .git",
			source: "https://github.com/Dolyyyy/huma_golang_api_template_modules.git",
			owner:  "Dolyyyy",
			repo:   "huma_golang_api_template_modules",
			ok:     true,
		},
		{
			name:   "git@",
			source: "git@github.com:Dolyyyy/huma_golang_api_template_modules.git",
			owner:  "Dolyyyy",
			repo:   "huma_golang_api_template_modules",
			ok:     true,
		},
		{
			name:   "ssh://git@",
			source: "ssh://git@github.com/Dolyyyy/huma_golang_api_template_modules.git",
			owner:  "Dolyyyy",
			repo:   "huma_golang_api_template_modules",
			ok:     true,
		},
		{
			name:   "invalid",
			source: "https://gitlab.com/Dolyyyy/huma_golang_api_template_modules",
			ok:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			owner, repo, ok := parseGitHubOwnerRepo(tc.source)
			if ok != tc.ok {
				t.Fatalf("unexpected ok state: got=%v want=%v", ok, tc.ok)
			}
			if owner != tc.owner {
				t.Fatalf("unexpected owner: got=%q want=%q", owner, tc.owner)
			}
			if repo != tc.repo {
				t.Fatalf("unexpected repo: got=%q want=%q", repo, tc.repo)
			}
		})
	}
}

func TestModulesIndexURLCandidatesForGitHubRepo(t *testing.T) {
	t.Parallel()

	candidates := modulesIndexURLCandidates("https://github.com/Dolyyyy/huma_golang_api_template_modules")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d (%v)", len(candidates), candidates)
	}

	if candidates[0] != "https://raw.githubusercontent.com/Dolyyyy/huma_golang_api_template_modules/main/modules.json" {
		t.Fatalf("unexpected first candidate: %s", candidates[0])
	}
	if candidates[1] != "https://raw.githubusercontent.com/Dolyyyy/huma_golang_api_template_modules/master/modules.json" {
		t.Fatalf("unexpected second candidate: %s", candidates[1])
	}
}

func TestListUsesRemoteModulesIndex(t *testing.T) {
	projectRoot := prepareProjectRoot(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/modules.json" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
  {
    "id": "auth-token",
    "name": "API token auth",
    "description": "Protect API routes using X-API-Token or Authorization: Bearer <token>."
  },
  {
    "id": "metrics-prometheus",
    "name": "Prometheus metrics",
    "description": "Expose /metrics for Prometheus scraping."
  }
]`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithRoot([]string{"--source", server.URL + "/modules.json", "list"}, projectRoot, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "modules source: "+server.URL+"/modules.json") {
		t.Fatalf("expected remote source in output, got:\n%s", output)
	}
	if strings.Contains(output, "templatectl-modules-") {
		t.Fatalf("expected no temp clone path in list output, got:\n%s", output)
	}
	if !strings.Contains(output, "auth-token [available]") {
		t.Fatalf("expected auth-token in output, got:\n%s", output)
	}
	if !strings.Contains(output, "metrics-prometheus [available]") {
		t.Fatalf("expected metrics-prometheus in output, got:\n%s", output)
	}
}
