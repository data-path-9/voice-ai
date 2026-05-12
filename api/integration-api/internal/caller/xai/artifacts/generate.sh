#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="${ROOT_DIR}/proto"
OUT_DIR="${ROOT_DIR}"

protoc \
  -I "${PROTO_DIR}" \
  --go_out="${OUT_DIR}" \
  --go-grpc_out="${OUT_DIR}" \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=Mxai/api/v1/auth.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/auth.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/chat.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/chat.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/deferred.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/deferred.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/documents.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/documents.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/image.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/image.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/sample.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/sample.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go_opt=Mxai/api/v1/usage.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  --go-grpc_opt=Mxai/api/v1/usage.proto=github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1 \
  xai/api/v1/auth.proto \
  xai/api/v1/chat.proto \
  xai/api/v1/deferred.proto \
  xai/api/v1/documents.proto \
  xai/api/v1/image.proto \
  xai/api/v1/sample.proto \
  xai/api/v1/usage.proto
