package runner

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Report struct {
	Result string `json:"result"`
}

func TestReadJsonFile_Errors(t *testing.T) {
	var got Report

	testCases := []struct {
		fileName    string
		wantErrorAs any
	}{
		{
			fileName:    "file_not_exist",
			wantErrorAs: new(*fs.PathError),
		},
		{
			fileName:    filepath.Join("..", "..", "test", "fixtures", "invalid_report.txt"),
			wantErrorAs: new(*json.SyntaxError),
		},
	}

	for _, tc := range testCases {
		err := readJsonFile(tc.fileName, &got)

		if got.Result != "" {
			t.Errorf("readJsonFile(%q, &got) = %s", tc.fileName, got)
		}

		if err != nil {
			if !errors.As(err, tc.wantErrorAs) {
				t.Errorf("readJsonFile(%q, &got) = %v, want %T", tc.fileName, err, tc.wantErrorAs)
			}
		}
	}
}

func TestReadJsonFile(t *testing.T) {
	var got Report
	fileName := filepath.Join("..", "..", "test", "fixtures", "report.json")
	want := Report{
		Result: "pass",
	}

	err := readJsonFile(fileName, &got)
	if err != nil {
		t.Errorf("readJsonFile(%q, &got) = %v", fileName, err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("readJsonFile(%s) diff (-got +want):\n%s", fileName, diff)
	}
}
