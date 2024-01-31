package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// TestCase represents a single test case.
type TestCase struct {
	Path              string `json:"path"`
	EstimatedDuration *int   `json:"estimated_duration"`
}

// Tests represents a set of tests.
type Tests struct {
	Cases  []TestCase `json:"cases"`
	Format string     `json:"format"`
}

type Task struct {
	NodeNumber int   `json:"node_number"`
	Tests      Tests `json:"tests"`
}

type TestPlan struct {
	Tasks map[string]Task `json:"tasks"`
}

type TestPlanParams struct {
	SuiteToken  string `json:"suite_token"`
	Mode        string `json:"mode"`
	Identifier  string `json:"identifier"`
	Parallelism int    `json:"parallelism"`
	Tests       Tests  `json:"tests"`
}

// FetchTestPlan fetches a test plan from the service.
func FetchTestPlan(splitterPath string, params TestPlanParams) TestPlan {
	// convert params to json string
	requestBody, err := json.Marshal(params)
	if err != nil {
		log.Fatal("Error when converting params to json string: ", err)
	}

	// create request
	postUrl := splitterPath + "/test-splitting/plan"
	r, err := http.NewRequest("POST", postUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatal("Error when creating request: ", err)
	}
	r.Header.Add("Content-Type", "application/json")

	// send request
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Fatal("Error when sending request: ", err)
	}
	defer resp.Body.Close()

	// TODO: check the response status code

	// read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error when reading response body: ", err)
	}

	// parse response
	var testPlan TestPlan
	err = json.Unmarshal(responseBody, &testPlan)
	if err != nil {
		log.Fatal("Error when parsing response: ", err)
	}

	return testPlan
}
