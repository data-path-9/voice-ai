// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"fmt"

	"github.com/emiago/sipgo"
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

func NewServer(ctx context.Context, cfg *ServerConfig) (*Server, error) {
	inner, err := internal_core.NewServer(ctx, cfg.toCore())
	if err != nil {
		return nil, err
	}
	return &Server{inner: inner}, nil
}

func (s *Server) Start() error {
	return s.inner.Start()
}

func (s *Server) Stop() {
	s.inner.Stop()
}

func (s *Server) SetConfigResolver(resolver ConfigResolver) {
	s.inner.SetConfigResolver(adaptConfigResolver(resolver))
}

func (s *Server) SetMiddlewares(middlewares []Middleware, final ConfigResolver) {
	s.SetConfigResolver(MiddlewareChain(middlewares, final))
}

func (s *Server) IsRunning() bool {
	return s.inner.IsRunning()
}

func (s *Server) AllocateRTPPort() (int, error) {
	return s.inner.AllocateRTPPort()
}

func (s *Server) ReleaseRTPPort(port int) {
	s.inner.ReleaseRTPPort(port)
}

func (s *Server) NegotiatedSDPConfig(localIP string, rtpPort int, codec *Codec) *SDPConfig {
	return sdpConfigFromCore(s.inner.NegotiatedSDPConfig(localIP, rtpPort, codec.toCore()))
}

func (s *Server) GenerateSDP(config *SDPConfig) string {
	return s.inner.GenerateSDP(config.toCore())
}

func (s *Server) ParseSDP(sdpBody []byte) (*SDPMediaInfo, error) {
	info, err := s.inner.ParseSDP(sdpBody)
	if err != nil {
		return nil, err
	}
	return sdpInfoFromCore(info), nil
}

func (s *Server) Client() *sipgo.Client {
	return s.inner.Client()
}

func (s *Server) GetListenConfig() *ListenConfig {
	return listenConfigFromCore(s.inner.GetListenConfig())
}

func (s *Server) SessionCount() int {
	return s.inner.SessionCount()
}

func (s *Server) SetOnApplicationReady(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnApplicationReady(nil)
		return
	}
	s.inner.SetOnApplicationReady(func(session *internal_core.Session, fromURI, toURI string) error {
		return fn(wrapSession(session), fromURI, toURI)
	})
}

func (s *Server) SetOnApplicationCleanup(fn func(session *Session)) {
	if fn == nil {
		s.inner.SetOnApplicationCleanup(nil)
		return
	}
	s.inner.SetOnApplicationCleanup(func(session *internal_core.Session) {
		fn(wrapSession(session))
	})
}

func (s *Server) SetOnInvite(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnInvite(nil)
		return
	}
	s.inner.SetOnInvite(func(session *internal_core.Session, fromURI, toURI string) error {
		return fn(wrapSession(session), fromURI, toURI)
	})
}

func (s *Server) SetOnBye(fn func(session *Session) error) {
	if fn == nil {
		s.inner.SetOnBye(nil)
		return
	}
	s.inner.SetOnBye(func(session *internal_core.Session) error {
		return fn(wrapSession(session))
	})
}

func (s *Server) SetOnCancel(fn func(session *Session) error) {
	if fn == nil {
		s.inner.SetOnCancel(nil)
		return
	}
	s.inner.SetOnCancel(func(session *internal_core.Session) error {
		return fn(wrapSession(session))
	})
}

func (s *Server) SetOnError(fn func(session *Session, err error)) {
	if fn == nil {
		s.inner.SetOnError(nil)
		return
	}
	s.inner.SetOnError(func(session *internal_core.Session, err error) {
		fn(wrapSession(session), err)
	})
}

func (s *Server) HealthSnapshot() ServerHealthSnapshot {
	return serverHealthSnapshotFromCore(s.inner.HealthSnapshot())
}

func adaptConfigResolver(resolver ConfigResolver) internal_core.ConfigResolver {
	if resolver == nil {
		return nil
	}
	return func(ctx *internal_core.SIPRequestContext) (*internal_core.InviteResult, error) {
		result, err := resolver(sipRequestContextFromCore(ctx))
		if err != nil {
			return nil, err
		}
		return inviteResultToCore(result), nil
	}
}

func sipRequestContextFromCore(ctx *internal_core.SIPRequestContext) *SIPRequestContext {
	if ctx == nil {
		return nil
	}
	return &SIPRequestContext{
		Method:      ctx.Method,
		CallID:      ctx.CallID,
		FromURI:     ctx.FromURI,
		ToURI:       ctx.ToURI,
		SDPInfo:     sdpInfoFromCore(ctx.SDPInfo),
		APIKey:      ctx.APIKey,
		AssistantID: ctx.AssistantID,
		Extra:       cloneMap(ctx.Extra),
	}
}

func inviteResultToCore(result *InviteResult) *internal_core.InviteResult {
	if result == nil {
		return nil
	}
	return &internal_core.InviteResult{
		Config:      result.Config.toCore(),
		ShouldAllow: result.ShouldAllow,
		RejectCode:  result.RejectCode,
		RejectMsg:   result.RejectMsg,
		Extra:       cloneMap(result.Extra),
	}
}
