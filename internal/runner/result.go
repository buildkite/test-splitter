package runner

type Status string

const (
	StatusPassed Status = "passed"
	StatusFailed Status = "failed"
	StatusError  Status = "error"
)

type Result struct {
	Status      Status
	FailedTests []string
}
