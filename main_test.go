package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsGenerated(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"foo.go", false},
		{"pkg/handler.go", false},
		{"internal/gen/types.go", true},
		{"foo_gen.go", true},
		{"api.pb.go", true},
		{"queries.sql.go", true},
		{"gen/foo.go", false}, // must have /gen/ not just gen/
		{"/gen/foo.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isGenerated(tt.path)
			if got != tt.want {
				t.Errorf("isGenerated(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestAnalyzeFiles(t *testing.T) {
	dir := t.TempDir()

	smallFile := filepath.Join(dir, "small.go")
	if err := os.WriteFile(smallFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	largeContent := make([]byte, 50000)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	largeFile := filepath.Join(dir, "large.go")
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{smallFile, largeFile}
	results, violations := analyzeFiles(files, 25000, 0.65)

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	if len(violations) != 1 {
		t.Errorf("got %d violations, want 1", len(violations))
	}

	if len(violations) > 0 && violations[0].path != largeFile {
		t.Errorf("expected large file to be a violation")
	}
}

func TestExpandArgs(t *testing.T) {
	dir := t.TempDir()

	files := []string{"a.go", "b.go", "c.txt"}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "d.go"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("single file", func(t *testing.T) {
		got, err := expandArgs([]string{filepath.Join(dir, "a.go")})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("got %d files, want 1", len(got))
		}
	})

	t.Run("directory non-recursive", func(t *testing.T) {
		got, err := expandArgs([]string{dir})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("got %d files, want 2 (.go files only)", len(got))
		}
	})

	t.Run("directory recursive", func(t *testing.T) {
		got, err := expandArgs([]string{dir + "/..."})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Errorf("got %d files, want 3", len(got))
		}
	})
}

func TestRunValidation(t *testing.T) {
	t.Run("negative ratio", func(t *testing.T) {
		code := run([]string{"-ratio", "-1", "."})
		if code != 1 {
			t.Errorf("expected exit code 1 for negative ratio, got %d", code)
		}
	})

	t.Run("zero threshold", func(t *testing.T) {
		code := run([]string{"-threshold", "0", "."})
		if code != 1 {
			t.Errorf("expected exit code 1 for zero threshold, got %d", code)
		}
	})

	t.Run("help flag", func(t *testing.T) {
		code := run([]string{"-h"})
		if code != 0 {
			t.Errorf("expected exit code 0 for help, got %d", code)
		}
	})
}

func TestRunWithFiles(t *testing.T) {
	dir := t.TempDir()

	smallFile := filepath.Join(dir, "ok.go")
	if err := os.WriteFile(smallFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("files under threshold", func(t *testing.T) {
		code := run([]string{smallFile})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	t.Run("files over threshold", func(t *testing.T) {
		code := run([]string{"-threshold", "1", smallFile})
		if code != 1 {
			t.Errorf("expected exit code 1 for violation, got %d", code)
		}
	})
}
