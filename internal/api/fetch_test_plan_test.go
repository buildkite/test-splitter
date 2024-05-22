package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buildkite/test-splitter/internal/plan"
	"github.com/google/go-cmp/cmp"
)

func TestFetchTestPlan(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
	"tasks": {
		"task_1": {
			"node_number": 1,
			"tests": [
				{
					"path": "hello_world_spec.rb",
					"format": "file"
				}
			]
		}
	}
}`)
	}))
	defer svr.Close()

	cfg := ClientConfig{
		AccessToken:      "asdf1234",
		OrganizationSlug: "my-org",
		ServerBaseUrl:    svr.URL,
	}

	c := NewClient(cfg)
	got, err := c.FetchTestPlan("my-suite", "xyz")

	want := plan.TestPlan{
		Tasks: map[string]*plan.Task{
			"task_1": {
				NodeNumber: 1,
				Tests: []plan.TestCase{{
					Path:   "hello_world_spec.rb",
					Format: "file",
				}},
			},
		},
	}

	if err != nil {
		t.Errorf("FetchTestPlan() error = %v", err)
	}

	if diff := cmp.Diff(got, &want); diff != "" {
		t.Errorf("FetchTestPlan() diff (-got +want):\n%s", diff)
	}
}

func TestFetchTestPlan_NotFound(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svr.Close()

	cfg := ClientConfig{
		AccessToken:      "asdf1234",
		OrganizationSlug: "my-org",
		ServerBaseUrl:    svr.URL,
	}

	c := NewClient(cfg)
	got, err := c.FetchTestPlan("my-suite", "xyz")

	if err != nil {
		t.Errorf("FetchTestPlan() error = %v", err)
	}

	if got != nil {
		t.Errorf("FetchTestPlan() = %v, want nil", got)
	}
}

func TestFetchTestPlan_InvalidRequest(t *testing.T) {
	statusCodes := []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("status code %d", code), func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer svr.Close()

			cfg := ClientConfig{
				AccessToken:      "asdf1234",
				OrganizationSlug: "my-org",
				ServerBaseUrl:    svr.URL,
			}

			c := NewClient(cfg)
			got, err := c.FetchTestPlan("my-suite", "xyz")

			if err == nil {
				t.Errorf("FetchTestPlan() want error, got nil")
			}

			if got != nil {
				t.Errorf("FetchTestPlan() = %v, want nil", got)
			}
		})
	}
}

func TestFetchTestPlan_ServerError(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer svr.Close()

	cfg := ClientConfig{
		AccessToken:      "asdf1234",
		OrganizationSlug: "my-org",
		ServerBaseUrl:    svr.URL,
	}

	c := NewClient(cfg)
	got, err := c.FetchTestPlan("my-suite", "xyz")

	if err != nil {
		t.Errorf("FetchTestPlan() error = %v", err)
	}

	if got != nil {
		t.Errorf("FetchTestPlan() = %v, want nil", got)
	}
}
