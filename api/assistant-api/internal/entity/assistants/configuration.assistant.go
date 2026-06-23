// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/utils"
)

type AssistantConfigurationType string

const (
	AssistantConfigurationTypeAuthentication AssistantConfigurationType = "authentication"
	AssistantConfigurationTypeWebhook        AssistantConfigurationType = "webhook"
	AssistantConfigurationTypeAnalysis       AssistantConfigurationType = "analysis"
	AssistantConfigurationTypeTelemetry      AssistantConfigurationType = "telemetry"
	AssistantConfigurationTypeStorage        AssistantConfigurationType = "storage"
)

type AssistantConfiguration struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	Enabled           bool                            `json:"enabled" gorm:"type:boolean;not null;default:true"`
	AssistantId       uint64                          `json:"assistantId" gorm:"type:bigint;size:20;not null"`
	ConfigurationType AssistantConfigurationType      `json:"configurationType" gorm:"type:varchar(50);not null"`
	Provider          string                          `json:"provider" gorm:"type:varchar(50);not null"`
	Options           []*AssistantConfigurationOption `json:"options" gorm:"foreignKey:AssistantConfigurationId"`
}

func (AssistantConfiguration) TableName() string {
	return "assistant_configurations"
}

func (a *AssistantConfiguration) GetOptions() utils.Option {
	opts := make(utils.Option, len(a.Options))
	for _, v := range a.Options {
		opts[v.Key] = v.Value
	}
	return opts
}

type AssistantConfigurationOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantConfigurationId uint64 `json:"assistantConfigurationId" gorm:"type:bigint;size:20;not null"`
}

func (AssistantConfigurationOption) TableName() string {
	return "assistant_configuration_options"
}
