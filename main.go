// The test-splitter tool fetches and runs test plans generated by Buildkite
// Test Splitting.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/buildkite/test-splitter/internal/api"
	"github.com/buildkite/test-splitter/internal/config"
	"github.com/buildkite/test-splitter/internal/debug"
	"github.com/buildkite/test-splitter/internal/plan"
	"github.com/buildkite/test-splitter/internal/runner"
	"golang.org/x/sys/unix"
)

var Version = ""

type TestRunner interface {
	Run(testCases []string, retry bool) (runner.TestResult, error)
	GetExamples(files []string) ([]plan.TestCase, error)
	GetFiles() ([]string, error)
	Name() string
}

func main() {
	debug.SetDebug(os.Getenv("BUILDKITE_SPLITTER_DEBUG_ENABLED") == "true")

	versionFlag := flag.Bool("version", false, "print version information")

	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	// get config
	cfg, err := config.New()
	if err != nil {
		logErrorAndExit(16, "Invalid configuration: %v", err)
	}

	testRunner, err := runner.DetectRunner(cfg)
	if err != nil {
		logErrorAndExit(16, "Unsupported value for BUILDKITE_SPLITTER_TEST_RUNNER")
	}

	files, err := testRunner.GetFiles()
	if err != nil {
		logErrorAndExit(16, "Couldn't get files: %v", err)
	}

	// get plan
	ctx := context.Background()
	apiClient := api.NewClient(api.ClientConfig{
		ServerBaseUrl:    cfg.ServerBaseUrl,
		AccessToken:      cfg.AccessToken,
		OrganizationSlug: cfg.OrganizationSlug,
		Version:          Version,
	})

	testPlan, err := fetchOrCreateTestPlan(ctx, apiClient, cfg, files, testRunner)
	if err != nil {
		logErrorAndExit(16, "Couldn't fetch or create test plan: %v", err)
	}

	debug.Printf("My favourite ice cream is %s", testPlan.Experiment)

	// get plan for this node
	thisNodeTask := testPlan.Tasks[strconv.Itoa(cfg.NodeIndex)]

	// execute tests
	runnableTests := []string{}
	for _, testCase := range thisNodeTask.Tests {
		runnableTests = append(runnableTests, testCase.Path)
	}

	var timeline []api.Timeline
	testResult, err := runTestsWithRetry(testRunner, runnableTests, cfg.MaxRetries, &timeline)

	if err != nil {
		if ProcessSignaledError := new(runner.ProcessSignaledError); errors.As(err, &ProcessSignaledError) {
			logSignalAndExit(testRunner.Name(), ProcessSignaledError.Signal)
		}

		if exitError := new(exec.ExitError); errors.As(err, &exitError) {
			sendMetadata(ctx, apiClient, cfg, timeline)
			logErrorAndExit(exitError.ExitCode(), "%s exited with error %v", testRunner.Name(), err)
		}

		logErrorAndExit(16, "Couldn't run tests: %v", err)
	}

	if testResult.Status == runner.TestStatusFailed {
		sendMetadata(ctx, apiClient, cfg, timeline)
		if failedCount := len(testResult.FailedTests); failedCount > 1 {
			logErrorAndExit(1, "%s exited with %d failures", testRunner.Name(), failedCount)
		}
		logErrorAndExit(1, "%s exited with 1 failure", testRunner.Name())
	}

	sendMetadata(ctx, apiClient, cfg, timeline)
}

func createTimestamp() string {
	return time.Now().Format(time.RFC3339Nano)
}

func sendMetadata(ctx context.Context, apiClient *api.Client, cfg config.Config, timeline []api.Timeline) {
	err := apiClient.PostTestPlanMetadata(ctx, cfg.SuiteSlug, cfg.Identifier, api.TestPlanMetadataParams{
		Timeline:    timeline,
		SplitterEnv: cfg.DumpEnv(),
		Version:     Version,
	})

	// Error is suppressed because we don't want to fail the build if we can't send metadata.
	if err != nil {
		fmt.Printf("Failed to send metadata to Test Analytics: %v\n", err)
	}
}

func runTestsWithRetry(testRunner TestRunner, testsCases []string, maxRetries int, timeline *[]api.Timeline) (runner.TestResult, error) {
	attemptCount := 0

	var testResult runner.TestResult
	var err error

	for attemptCount <= maxRetries {
		if attemptCount == 0 {
			fmt.Printf("+++ Buildkite Test Splitter: Running tests\n")
			*timeline = append(*timeline, api.Timeline{
				Event:     "test_start",
				Timestamp: createTimestamp(),
			})
		} else {
			fmt.Printf("+++ Buildkite Test Splitter: ♻️ Attempt %d of %d to retry failing tests\n", attemptCount, maxRetries)
			*timeline = append(*timeline, api.Timeline{
				Event:     fmt.Sprintf("retry_%d_start", attemptCount),
				Timestamp: createTimestamp(),
			})
		}

		testResult, err = testRunner.Run(testsCases, false)

		if attemptCount == 0 {
			*timeline = append(*timeline, api.Timeline{
				Event:     "test_end",
				Timestamp: createTimestamp(),
			})
		} else {
			*timeline = append(*timeline, api.Timeline{
				Event:     fmt.Sprintf("retry_%d_end", attemptCount),
				Timestamp: createTimestamp(),
			})
		}

		// Don't retry if we've reached max retries.
		if attemptCount == maxRetries {
			break
		}

		// Don't retry if there is an error that is not a test failure.
		if err != nil {
			break
		}

		// Don't retry if tests are passed.
		if testResult.Status == runner.TestStatusPassed {
			break
		}

		// Retry only the failed tests.
		testsCases = testResult.FailedTests
		attemptCount++
	}

	return testResult, err
}

func logSignalAndExit(name string, signal syscall.Signal) {
	fmt.Printf("Buildkite Test Splitter: %s was terminated with signal: %v (%v)\n", name, unix.SignalName(signal), signal)

	exitCode := 128 + int(signal)
	os.Exit(exitCode)
}

// logErrorAndExit logs an error message and exits with the given exit code.
func logErrorAndExit(exitCode int, format string, v ...any) {
	fmt.Printf("Buildkite Test Splitter: "+format+"\n", v...)
	os.Exit(exitCode)
}

// fetchOrCreateTestPlan fetches a test plan from the server, or creates a
// fallback plan if the server is unavailable or returns an error plan.
func fetchOrCreateTestPlan(ctx context.Context, apiClient *api.Client, cfg config.Config, files []string, testRunner TestRunner) (plan.TestPlan, error) {
	debug.Println("Fetching test plan")

	// Fetch the plan from the server's cache.
	cachedPlan, err := apiClient.FetchTestPlan(ctx, cfg.SuiteSlug, cfg.Identifier)

	handleError := func(err error) (plan.TestPlan, error) {
		if errors.Is(err, api.ErrRetryTimeout) {
			fmt.Println("Could not fetch or create plan from server, using fallback mode. Your build may take longer than usual.")
			p := plan.CreateFallbackPlan(files, cfg.Parallelism)
			return p, nil
		}
		return plan.TestPlan{}, err
	}

	if err != nil {
		return handleError(err)
	}

	if cachedPlan != nil {
		// The server can return an "error" plan indicated by an empty task list (i.e. `{"tasks": {}}`).
		// In this case, we should create a fallback plan.
		if len(cachedPlan.Tasks) == 0 {
			fmt.Println("Error plan received, using fallback mode. Your build may take longer than usual.")
			testPlan := plan.CreateFallbackPlan(files, cfg.Parallelism)
			return testPlan, nil
		}

		debug.Printf("Test plan found. Identifier: %q", cfg.Identifier)
		return *cachedPlan, nil
	}

	debug.Println("No test plan found, creating a new plan")
	// If the cache is empty, create a new plan.
	params, err := createRequestParam(ctx, cfg, files, *apiClient, testRunner)
	if err != nil {
		return handleError(err)
	}

	debug.Println("Creating test plan")
	testPlan, err := apiClient.CreateTestPlan(ctx, cfg.SuiteSlug, params)

	if err != nil {
		return handleError(err)
	}

	// The server can return an "error" plan indicated by an empty task list (i.e. `{"tasks": {}}`).
	// In this case, we should create a fallback plan.
	if len(testPlan.Tasks) == 0 {
		fmt.Println("Error plan received, using fallback mode. Your build may take longer than usual.")
		testPlan = plan.CreateFallbackPlan(files, cfg.Parallelism)
	}

	debug.Printf("Test plan created. Identifier: %q", cfg.Identifier)
	return testPlan, nil
}

type fileTiming struct {
	Path     string
	Duration time.Duration
}

// createRequestParam creates the request parameters for the test plan with the given configuration and files.
// The files should have been filtered by include/exclude patterns before passing to this function.
// If SplitByExample is disabled (default), it will return the default params that contain all the files.
// If SplitByExample is enabled, it will split the slow files into examples and return it along with the rest of the files.
//
// Error is returned if there is a failure to fetch test file timings or to get the test examples from test files when SplitByExample is enabled.
func createRequestParam(ctx context.Context, cfg config.Config, files []string, client api.Client, runner TestRunner) (api.TestPlanParams, error) {
	if !cfg.SplitByExample {
		debug.Println("Splitting by file")
		testCases := []plan.TestCase{}
		for _, file := range files {
			testCases = append(testCases, plan.TestCase{
				Path: file,
			})
		}

		return api.TestPlanParams{
			Identifier:  cfg.Identifier,
			Parallelism: cfg.Parallelism,
			Branch:      cfg.Branch,
			Tests: api.TestPlanParamsTest{
				Files: testCases,
			},
		}, nil
	}

	debug.Println("Splitting by example")

	debug.Printf("Fetching timings for %d files", len(files))
	// Fetch the timings for all files.
	timings, err := client.FetchFilesTiming(ctx, cfg.SuiteSlug, files)
	if err != nil {
		return api.TestPlanParams{}, fmt.Errorf("failed to fetch file timings: %w", err)
	}
	debug.Printf("Got timings for %d files", len(timings))

	// The server only returns timings for the files that has been run before.
	// Therefore, we need to merge the response with the requested files.
	// The files that are not in the response will have a duration of 0.
	allFilesTiming := []fileTiming{}
	for _, file := range files {
		allFilesTiming = append(allFilesTiming, fileTiming{
			Path:     file,
			Duration: timings[file],
		})
	}

	// Get files that has duration greater or equal to the slow file threshold.
	// Currently, the slow file threshold is set to 3 minutes which is roughly 70% of optimal 4 minutes node duration.
	slowFiles := []string{}
	restOfFiles := []plan.TestCase{}

	for _, timing := range allFilesTiming {
		if timing.Duration >= cfg.SlowFileThreshold {
			slowFiles = append(slowFiles, timing.Path)
		} else {
			restOfFiles = append(restOfFiles, plan.TestCase{
				Path: timing.Path,
			})
		}
	}

	if len(slowFiles) == 0 {
		debug.Println("No slow files found")
		return api.TestPlanParams{
			Identifier:  cfg.Identifier,
			Parallelism: cfg.Parallelism,
			Branch:      cfg.Branch,
			Tests: api.TestPlanParamsTest{
				Files: restOfFiles,
			},
		}, nil
	}

	debug.Printf("Getting examples for %d slow files", len(slowFiles))

	// Get the examples for the slow files.
	slowFilesExamples, err := runner.GetExamples(slowFiles)
	if err != nil {
		return api.TestPlanParams{}, fmt.Errorf("failed to get examples for slow files: %w", err)
	}

	debug.Printf("Got %d examples within the slow files", len(slowFilesExamples))

	return api.TestPlanParams{
		Identifier:  cfg.Identifier,
		Parallelism: cfg.Parallelism,
		Branch:      cfg.Branch,
		Tests: api.TestPlanParamsTest{
			Examples: slowFilesExamples,
			Files:    restOfFiles,
		},
	}, nil
}
