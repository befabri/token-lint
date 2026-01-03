# token-lint

A linter that checks if Go files exceed a token limit, helping keep files readable by LLMs.

## Installation

```bash
go install github.com/befabri/token-lint@latest
```

Or add as a tool dependency in your `go.mod`:

```bash
go get -tool github.com/befabri/token-lint@latest
```

## Usage

```bash
# Check all Go files recursively
tokenlint ./...

# Check specific file or directory
tokenlint path/to/file.go
tokenlint path/to/dir/...

# Show all files sorted by token count
tokenlint -all ./...

# Custom threshold (default: 25000)
tokenlint -threshold 20000 ./...

# Custom tokens-per-character ratio
tokenlint -ratio 0.65 ./...
```

## How it works

The tool estimates token counts using a character-based ratio calibrated for Claude's tokenizer on Go code (~0.65 tokens per character). This provides a fast approximation without requiring external tokenizer dependencies.

Files matching these patterns are skipped by default:
- `/gen/` directories
- `*_gen.go` files
- `*.pb.go` (protobuf)
- `*.sql.go` (sqlc)

## Exit codes

- `0` - All files under threshold
- `1` - One or more files exceed threshold

## Example output

```
$ tokenlint -all ./...
FILE                                         TOKENS    CHARS
--------------------------------------------------------------
pkg/server/handler.go                         32000    49230 <- EXCEEDS LIMIT
pkg/api/client.go                             18500    28461
pkg/utils/helpers.go                           8200    12615
...

1 file(s) exceed 25000 token threshold:

  pkg/server/handler.go
    ~32000 tokens (128% of limit, 49230 chars)
    Consider splitting into smaller files for better LLM readability
```

## License

MIT
