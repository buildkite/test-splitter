package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buildkite/test-splitter/internal/plan"
	"github.com/google/go-cmp/cmp"
)

func ptr[T any](t T) *T { return &t }

func TestFetchTestPlan(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
	"tasks": {
		"task_1": {
			"node_number": 1,
			"tests": {
				"cases": [
					{
						"path": "dummy.spec",
						"estimated_duration": 1000000
					}
				],
				"format": "junit"
			}
		}
	}
}`)
	}))
	defer svr.Close()

	ctx := context.Background()

	params := TestPlanParams{}
	got, err := FetchTestPlan(ctx, svr.URL, params)
	if err != nil {
		t.Errorf("FetchTestPlan(%q, %v) error = %v", svr.URL, params, err)
	}
	want := plan.TestPlan{
		Tasks: map[string]*plan.Task{
			"task_1": {
				NodeNumber: 1,
				Tests: plan.Tests{
					Cases: []plan.TestCase{{
						Path:              "dummy.spec",
						EstimatedDuration: ptr(1_000_000),
					}},
					Format: "junit",
				},
			},
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("FetchTestPlan(%q, %v) diff (-got +want):\n%s", svr.URL, params, diff)
	}
}

func TestFetchTestPlan_Error4xx(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer svr.Close()

	ctx := context.Background()
	params := TestPlanParams{}
	got, err := FetchTestPlan(ctx, svr.URL, params)

	wantTestPlan := plan.TestPlan{}

	if err == nil {
		t.Errorf("FetchTestPlan(%q, %v) should return an error", svr.URL, params)
	}

	if diff := cmp.Diff(got, wantTestPlan); diff != "" {
		t.Errorf("FetchTestPlan(%q, %v) diff (-got +want):\n%s", svr.URL, params, diff)
	}

	if !errors.Is(err, errInvalidRequest) {
		t.Errorf("FetchTestPlan(%q, %v) want %v", svr.URL, params, errInvalidRequest)
	}
}
