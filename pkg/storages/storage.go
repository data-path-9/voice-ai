// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package storages

import (
	"context"
	"time"

	"github.com/rapidaai/pkg/configs"
)

const FileWriteTimeout = 45 * time.Second

// such as Store or GetUrl.
type StorageOutput struct {
	// CompletePath is the fully resolved path or identifier
	// where the object is stored (local path, S3 key, GCS URL, etc.).
	CompletePath string `json:"complete_path"`

	// StorageType indicates which storage backend was used
	// (e.g., Local, S3, GCS).
	StorageType configs.StorageType `json:"storage_type"`

	// Error contains any error that occurred during the operation.
	// It is nil if the operation succeeded.
	Error error `json:"error"`
}

// GetStorageOutput represents the result of fetching data
// from a storage backend.
type GetStorageOutput struct {
	// Data contains the raw bytes of the stored object.
	Data []byte

	// Error contains any error that occurred while fetching the data.
	// It is nil if the operation succeeded.
	Error error
}

// Storage defines a generic interface for different storage backends
// (e.g., local filesystem, cloud storage, object storage).
//
// Implementations of this interface should be safe for concurrent use
// unless explicitly documented otherwise.
type Storage interface {
	// Name returns a human-readable name of the storage backend.
	// Example: "local", "s3", "gcs".
	Name() string

	// Store saves the given file content under the provided key.
	//
	// Parameters:
	//   - ctx: context for cancellation, timeout, and tracing
	//   - key: logical identifier or path for the stored object
	//   - fileContent: raw bytes to be stored
	//
	// Returns:
	//   - StorageOutput containing the final storage path and any error.
	Store(ctx context.Context, key string, fileContent []byte) StorageOutput

	// Get retrieves the stored object associated with the given key.
	//
	// Parameters:
	//   - ctx: context for cancellation, timeout, and tracing
	//   - key: logical identifier or path of the stored object
	//
	// Returns:
	//   - GetStorageOutput containing the raw data and any error.
	Get(ctx context.Context, key string) GetStorageOutput

	// GetUrl returns a URL or accessible path for the stored object.
	//
	// This is typically used when the storage backend can expose
	// a public or signed URL (e.g., S3 pre-signed URL).
	//
	// Parameters:
	//   - ctx: context for cancellation, timeout, and tracing
	//   - key: logical identifier or path of the stored object
	//
	// Returns:
	//   - StorageOutput containing the URL/path and any error.
	GetUrl(ctx context.Context, key string) StorageOutput
}
