# Remote File Fetcher Design

## Summary

Add the ability to fetch traffic files from remote storage (S3, etc.) using vals-style URI scheme (`ref+s3://bucket/key`). Extensible interface for adding more providers later.

## URI Convention

Follows helmfile/vals URI format:
- `ref+s3://bucket/path/to/file.gor` — AWS S3
- `ref+azureblob://container/path/to/file.gor` — Azure Blob (future)
- `./local/path/file.gor` — local file (no prefix)

## Architecture

### Interface

```go
type Fetcher interface {
    Fetch(uri string) (localPath string, cleanup func(), err error)
}
```

- `localPath`: path to a local file ready to be read
- `cleanup`: function to call when done (deletes temp file for remote, no-op for local)

### Implementations

| Fetcher | URI prefix | Auth |
|---------|-----------|------|
| `LocalFetcher` | no prefix / relative/absolute path | N/A |
| `S3Fetcher` | `ref+s3://` | AWS SDK default chain (env, ~/.aws/credentials, instance profile) |

### Factory

`fetcher.Resolve(uri string) (localPath string, cleanup func(), err error)` — detects scheme, delegates to appropriate fetcher.

## Integration

In `cmd/analyze.go`, before creating the traffic reader:

```go
localPath, cleanup, err := fetcher.Resolve(gorFile)
if err != nil { return err }
defer cleanup()
// use localPath instead of gorFile from here
```

## File Structure

| File | Role |
|------|------|
| `internal/fetcher/fetcher.go` | Interface + Resolve factory |
| `internal/fetcher/local.go` | LocalFetcher (passthrough) |
| `internal/fetcher/s3.go` | S3Fetcher (download to temp) |
| `internal/fetcher/fetcher_test.go` | Tests for local + factory |
| `cmd/analyze.go` | Call fetcher.Resolve before parser |
