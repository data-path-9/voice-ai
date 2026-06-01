// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"fmt"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
	"github.com/redis/go-redis/v9"
)

type ServerState int32

const (
	ServerStateCreated ServerState = iota
	ServerStateRunning
	ServerStateStopped
)

type SIPRequestContext struct {
	Method      string
	CallID      string
	FromURI     string
	ToURI       string
	SDPInfo     *SDPMediaInfo
	APIKey      string
	AssistantID string
	Extra       map[string]interface{}
}

func (c *SIPRequestContext) Set(key string, value interface{}) {
	if c.Extra == nil {
		c.Extra = make(map[string]interface{})
	}
	c.Extra[key] = value
}

func (c *SIPRequestContext) Get(key string) (interface{}, bool) {
	if c.Extra == nil {
		return nil, false
	}
	v, ok := c.Extra[key]
	return v, ok
}

type InviteResult struct {
	Config      *Config
	ShouldAllow bool
	RejectCode  int
	RejectMsg   string
	Extra       map[string]interface{}
}

func Reject(code int, msg string) *InviteResult {
	return &InviteResult{ShouldAllow: false, RejectCode: code, RejectMsg: msg}
}

func Allow(config *Config) *InviteResult {
	return &InviteResult{ShouldAllow: true, Config: config}
}

func AllowWithExtra(config *Config, extra map[string]interface{}) *InviteResult {
	return &InviteResult{ShouldAllow: true, Config: config, Extra: extra}
}

type ConfigResolver func(ctx *SIPRequestContext) (*InviteResult, error)

type Middleware func(ctx *SIPRequestContext, next func() (*InviteResult, error)) (*InviteResult, error)

func MiddlewareChain(middlewares []Middleware, final ConfigResolver) ConfigResolver {
	return func(ctx *SIPRequestContext) (*InviteResult, error) {
		var run func(i int) (*InviteResult, error)
		run = func(i int) (*InviteResult, error) {
			if i >= len(middlewares) {
				return final(ctx)
			}
			return middlewares[i](ctx, func() (*InviteResult, error) {
				return run(i + 1)
			})
		}
		return run(0)
	}
}

type Server struct {
	inner *internal_core.Server
}

type ListenConfig struct {
	Address                 string    `json:"address" mapstructure:"address"`
	ExternalIP              string    `json:"external_ip" mapstructure:"external_ip"`
	AllowLoopbackExternalIP bool      `json:"allow_loopback_external_ip" mapstructure:"allow_loopback_external_ip"`
	Port                    int       `json:"port" mapstructure:"port"`
	Transport               Transport `json:"transport" mapstructure:"transport"`
}

func (c *ListenConfig) GetExternalIP() string {
	if c == nil {
		return ""
	}
	if c.ExternalIP != "" {
		return c.ExternalIP
	}
	return c.Address
}

func (c *ListenConfig) GetBindAddress() string {
	if c == nil {
		return ""
	}
	return c.Address
}

func (c *ListenConfig) GetListenAddr() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

func (c *ListenConfig) toCore() *internal_core.ListenConfig {
	if c == nil {
		return nil
	}
	return &internal_core.ListenConfig{
		Address:                 c.Address,
		ExternalIP:              c.ExternalIP,
		AllowLoopbackExternalIP: c.AllowLoopbackExternalIP,
		Port:                    c.Port,
		Transport:               internal_core.Transport(c.Transport),
	}
}

func listenConfigFromCore(config *internal_core.ListenConfig) *ListenConfig {
	if config == nil {
		return nil
	}
	return &ListenConfig{
		Address:                 config.Address,
		ExternalIP:              config.ExternalIP,
		AllowLoopbackExternalIP: config.AllowLoopbackExternalIP,
		Port:                    config.Port,
		Transport:               Transport(config.Transport),
	}
}

type ServerConfig struct {
	ListenConfig      *ListenConfig
	ConfigResolver    ConfigResolver
	Logger            commons.Logger
	RedisClient       *redis.Client
	RTPPortRangeStart int
	RTPPortRangeEnd   int
}

func (c *ServerConfig) Validate() error {
	return c.toCore().Validate()
}

func (c *ServerConfig) toCore() *internal_core.ServerConfig {
	if c == nil {
		return nil
	}
	return &internal_core.ServerConfig{
		ListenConfig:      c.ListenConfig.toCore(),
		ConfigResolver:    adaptConfigResolver(c.ConfigResolver),
		Logger:            c.Logger,
		RedisClient:       c.RedisClient,
		RTPPortRangeStart: c.RTPPortRangeStart,
		RTPPortRangeEnd:   c.RTPPortRangeEnd,
	}
}
