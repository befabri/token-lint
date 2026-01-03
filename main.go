// token-lint checks if Go files exceed a token limit.
//
// Usage:
//
//	token-lint [flags] [files...]
//	token-lint ./...                    # Check all Go files recursively
//	token-lint -threshold 20000 file.go # Custom threshold
//
// Exit codes:
//
//	0 - All files under threshold
//	1 - One or more files exceed threshold
//
// Token estimation uses a character-based ratio calibrated for Claude's tokenizer
// on Go code (~0.65 tokens per character). Actual token counts may vary slightly.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultThreshold = 25000
	defaultRatio     = 0.65
)

type fileResult struct {
	path   string
	tokens int
	chars  int
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("token-lint", flag.ContinueOnError)
	threshold := fs.Int("threshold", defaultThreshold, "maximum tokens before warning")
	showAll := fs.Bool("all", false, "show token counts for all files, not just violations")
	ratio := fs.Float64("ratio", defaultRatio, "tokens per character ratio")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *ratio <= 0 {
		fmt.Fprintln(os.Stderr, "error: ratio must be positive")
		return 1
	}
	if *threshold <= 0 {
		fmt.Fprintln(os.Stderr, "error: threshold must be positive")
		return 1
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"./..."}
	}

	files, err := expandArgs(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no Go files found")
		return 0
	}

	results, violations := analyzeFiles(files, *threshold, *ratio)

	sort.Slice(results, func(i, j int) bool {
		return results[i].tokens > results[j].tokens
	})

	if *showAll {
		printAllResults(results, *threshold)
	}

	if len(violations) > 0 {
		printViolations(violations, *threshold)
		return 1
	}

	if !*showAll {
		fmt.Printf("All %d files under %d token threshold\n", len(results), *threshold)
	}
	return 0
}

func analyzeFiles(files []string, threshold int, ratio float64) ([]fileResult, []fileResult) {
	var results, violations []fileResult

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			continue
		}

		chars := len(content)
		tokens := int(float64(chars) * ratio)
		r := fileResult{path: path, tokens: tokens, chars: chars}
		results = append(results, r)

		if tokens > threshold {
			violations = append(violations, r)
		}
	}

	return results, violations
}

func printAllResults(results []fileResult, threshold int) {
	fmt.Printf("%-60s %8s %8s\n", "FILE", "TOKENS", "CHARS")
	fmt.Println(strings.Repeat("-", 78))
	for _, r := range results {
		marker := ""
		if r.tokens > threshold {
			marker = " <- EXCEEDS LIMIT"
		}
		fmt.Printf("%-60s %8d %8d%s\n", r.path, r.tokens, r.chars, marker)
	}
	fmt.Println()
}

func printViolations(violations []fileResult, threshold int) {
	fmt.Printf("%d file(s) exceed %d token threshold:\n\n", len(violations), threshold)
	for _, v := range violations {
		pct := float64(v.tokens) / float64(threshold) * 100
		fmt.Printf("  %s\n", v.path)
		fmt.Printf("    ~%d tokens (%.0f%% of limit, %d chars)\n", v.tokens, pct, v.chars)
		fmt.Printf("    Consider splitting into smaller files for better LLM readability\n\n")
	}
}

func expandArgs(args []string) ([]string, error) {
	var files []string

	for _, arg := range args {
		if arg == "./..." {
			// Recursively find all .go files
			err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(path, ".go") && !isGenerated(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if dir, ok := strings.CutSuffix(arg, "/..."); ok {
			// Recursively find .go files in directory
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(path, ".go") && !isGenerated(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if info, err := os.Stat(arg); err == nil && info.IsDir() {
			// Find .go files in directory (non-recursive)
			entries, err := os.ReadDir(arg)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					files = append(files, filepath.Join(arg, e.Name()))
				}
			}
		} else {
			// Single file
			files = append(files, arg)
		}
	}

	return files, nil
}

// isGenerated returns true for paths that contain generated code
func isGenerated(path string) bool {
	return strings.Contains(path, "/gen/") ||
		strings.Contains(path, "_gen.go") ||
		strings.HasSuffix(path, ".pb.go") ||
		strings.HasSuffix(path, ".sql.go")
}
