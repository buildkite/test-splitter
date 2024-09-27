package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

	if ProcessSignaledError := new(ProcessSignaledError); errors.As(err, &ProcessSignaledError) {
		return RunResult{Status: RunStatusError}, err
	}

	if exitError := new(exec.ExitError); errors.As(err, &exitError) {
		report, parseErr := p.parseReport(p.ResultPath)
		if parseErr != nil {
			fmt.Println("Buildkite Test Engine Client: Failed to read Playwright output, tests will not be retried.")
			return RunResult{Status: RunStatusError}, err
		}

		if report.Stats.Unexpected > 0 {
			var failedTests []string
			for _, suite := range report.Suites {
				for _, spec := range suite.Specs {
					if !spec.Ok {
						failedTests = append(failedTests, fmt.Sprintf("%s:%d", spec.File, spec.Line))
					}
				}
			}
			return RunResult{Status: RunStatusFailed, FailedTests: failedTests}, nil
		}
	}

	return RunResult{Status: RunStatusError}, err
}

func (p Playwright) parseReport(path string) (PlaywrightReport, error) {
	var report PlaywrightReport
	data, err := os.ReadFile(path)
	if err != nil {
		return PlaywrightReport{}, fmt.Errorf("failed to read playwright output: %v", err)
	}

	if err := json.Unmarshal(data, &report); err != nil {
		return PlaywrightReport{}, fmt.Errorf("failed to parse playwright output: %s", err)
	}

	return report, nil
}

func (p Playwright) GetFiles() ([]string, error) {
	return nil, nil
}

func (p Playwright) GetExamples(files []string) ([]plan.TestCase, error) {
	return nil, nil
}

type PlaywrightSpec struct {
	File   string
	Line   int
	Column int
	Id     string
	Title  string
	Ok     bool
}

type PlaywrightReportSuite struct {
	Title string
	Specs []PlaywrightSpec
}

type PlaywrightReport struct {
	Suites []PlaywrightReportSuite
	Stats  struct {
		Expected   int
		Unexpected int
	}
}
