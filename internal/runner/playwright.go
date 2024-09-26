package runner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/buildkite/test-engine-client/internal/plan"
)

var _ = TestRunner(Playwright{})

type Playwright struct {
	RunnerConfig
}

func (p Playwright) Name() string {
	return "Playwright"
}

func NewPlaywright(p RunnerConfig) Playwright {
	if p.TestCommand == "" {
		p.TestCommand = "playwright test"
	}

	if p.ResultPath == "" {
		p.ResultPath = "playwright.json"
	}

	return Playwright{p}
}

func (p Playwright) Run(testCases []string, retry bool) (RunResult, error) {
	cmdName := "yarn"
	cmdArgs := []string{"run", "playwright", "test"}
	cmdArgs = append(cmdArgs, testCases...)
	cmd := exec.Command(cmdName, cmdArgs...)

	fmt.Printf("%s %s\n", cmdName, strings.Join(cmdArgs, " "))
	err := runAndForwardSignal(cmd)

	if err == nil { // note: returning success early
		return RunResult{Status: RunStatusPassed}, nil
	}

	return RunResult{}, nil
}

func (p Playwright) GetFiles() ([]string, error) {
	return nil, nil
}

func (p Playwright) GetExamples(files []string) ([]plan.TestCase, error) {
	return nil, nil
}
