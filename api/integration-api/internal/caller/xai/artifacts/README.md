## xAI Artifacts

This directory contains committed protobuf artifacts for the xAI gRPC chat integration.

- Upstream repository: `https://github.com/xai-org/xai-proto.git`
- Upstream commit: `c9345abd85649ecb5b27fe8708f573b4ab9d6971`
- Source proto path: `proto/xai/api/v1`

### Included protos

- `auth.proto`
- `chat.proto`
- `deferred.proto`
- `documents.proto`
- `image.proto`
- `sample.proto`
- `usage.proto`

### Regeneration

Run:

```bash
./api/integration-api/internal/caller/xai/artifacts/generate.sh
```

This regenerates `artifacts/xai/api/v1/*.pb.go` from the committed proto sources.
