// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import gorm_model "github.com/rapidaai/pkg/models/gorm"

// AssistantHTTPLog stores generic HTTP interaction logs emitted by webhook,
// authentication, and analysis executors.
type AssistantHTTPLog struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational

	Source                  string  `json:"source" gorm:"type:string;size:50;not null"`
	SourceRefId             uint64  `json:"sourceRefId" gorm:"column:source_ref_id;type:bigint;not null;default:0"`
	SourceEvent             string  `json:"sourceEvent" gorm:"column:source_event;type:string;size:200;not null"`
	ContextId               string  `json:"contextId" gorm:"column:context_id;type:string;size:100"`
	AssistantId             uint64  `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantConversationId *uint64 `json:"assistantConversationId" gorm:"type:bigint"`
	HttpMethod              string  `json:"httpMethod" gorm:"type:string;size:200;not null"`
	HttpUrl                 string  `json:"httpUrl" gorm:"type:string;size:400;not null"`
	AssetPrefix             string  `json:"assetPrefix" gorm:"type:string;size:200;not null"`
	ResponseStatus          int64   `json:"responseStatus" gorm:"type:bigint;size:10"`
	TimeTaken               int64   `json:"timeTaken" gorm:"type:bigint;size:20"`
	RetryCount              uint32  `json:"retryCount" gorm:"type:bigint;size:20"`
	ErrorMessage            *string `json:"errorMessage" gorm:"type:text"`
}

func (AssistantHTTPLog) TableName() string {
	return "assistant_http_logs"
}
