package runner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlaywrightRun(t *testing.T) {
	mockCwd(t, "./fixtures/playwright")

	playwright := NewPlaywright(RunnerConfig{
		TestCommand: "yarn run playwright test",
		ResultPath:  "playwright.json",
	})

	files := []string{"./fixtures/playwright/tests/example.spec.js"}
	got, err := playwright.Run(files, false)

	want := RunResult{
		Status: RunStatusPassed,
	}

	if err != nil {
		t.Errorf("Playwright.Run(%q) error = %v", files, err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Playwright.Run(%q) diff (-got +want):\n%s", files, diff)
	}
}
