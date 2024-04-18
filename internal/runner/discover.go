package runner

import (
	"fmt"
	"io/fs"

	"github.com/DrJosh9000/zzglob"
)

func discoverTestFiles(pattern string, excludePattern string) []string {
	parsedPattern, err := zzglob.Parse(pattern)
	if err != nil {
		fmt.Printf("Error parsing pattern: %v\n", err)
	}

	parsedExcludePattern, err := zzglob.Parse(excludePattern)
	if err != nil {
		fmt.Printf("Error parsing exclude pattern: %v\n", err)
	}

	discoveredFiles := []string{}

	// Use the Glob function to traverse the directory recursively
	// and append the matched file paths to the discoveredFiles slice
	parsedPattern.Glob(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error walking: %v\n", err)
		}
		if d.IsDir() {
			return nil
		}

		// Check if the path matches the exclude pattern and skip it
		if parsedExcludePattern.Match(path) {
			return nil
		}

		discoveredFiles = append(discoveredFiles, path)
		return nil
	}, nil)

	return discoveredFiles
}