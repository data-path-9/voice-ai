// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "strconv"

type ErrorCode uint64

type PlatformError struct {
	HTTPStatusCode int
	Code           ErrorCode
	Error          string
	ErrorMessage   string
}

func (platformError PlatformError) HTTPStatusCodeInt32() int32 {
	return int32(platformError.HTTPStatusCode)
}

func (platformError PlatformError) CodeString() string {
	return strconv.FormatUint(uint64(platformError.Code), 10)
}
