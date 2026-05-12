// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_common

import "time"

type ClientConfig struct {
	Compatibility Compatibility
	BaseURL       string
	Headers       map[string]string

	ClientTimeout  time.Duration
	RequestTimeout time.Duration
}
