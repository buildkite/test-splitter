package plan

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

// Task represents the task for the given node.
type Task struct {
	NodeNumber int   `json:"node_number"`
	Tests      Tests `json:"tests"`
}

// TestPlan represents the entire test plan.
type TestPlan struct {
	Tasks map[string]Task `json:"tasks"`
}

// TestPlanParams represents the config params sent when fetching a test plan.
type TestPlanParams struct {
	SuiteToken  string `json:"suite_token"`
	Mode        string `json:"mode"`
	Identifier  string `json:"identifier"`
	Parallelism int    `json:"parallelism"`
	Tests       Tests  `json:"tests"`
}