# Remote File Fetching

wafflex can fetch traffic files from remote storage using URI schemes in the `traffic.file` config field.

## Supported Schemes

| Scheme | Provider | Example |
|---|---|---|
| `ref+s3://` | AWS S3 | `ref+s3://my-bucket/captures/traffic.gor` |
| (none) | Local file | `./traffic.gor` |

More providers (Azure Blob, GCS) can be added by implementing the `Fetcher` interface.

## Usage

### In Config

```yaml
traffic:
  type: gor
  file: ref+s3://my-bucket/captures/traffic.gor
```

### Via CLI Flag

```bash
wafflex analyze -g ref+s3://my-bucket/captures/traffic.gor -c coraza.conf
```

## Authentication

The S3 fetcher uses the default AWS SDK credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM instance profile (EC2, ECS)

## How It Works

1. The URI is parsed to detect the scheme
2. The file is downloaded to a temporary local file
3. The traffic reader processes the local file
4. The temp file is cleaned up automatically after analysis

## Adding a New Provider

Implement the `Fetcher` interface in `internal/fetcher/`:

```go
type Fetcher interface {
    Fetch(uri string) (localPath string, cleanup func(), err error)
}
```

Then register the new URI prefix in `Resolve()` in `internal/fetcher/fetcher.go`.
