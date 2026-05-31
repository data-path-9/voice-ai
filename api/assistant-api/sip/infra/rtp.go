// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"net"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
	"github.com/redis/go-redis/v9"
)

type RTPPacket struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CC             uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Payload        []byte
}

type RTPHandlerFactory func(context.Context, *RTPConfig) (*RTPHandler, error)

type RTPHandler struct {
	inner *internal_core.RTPHandler

	codec        *Codec
	audioInChan  chan []byte
	audioOutChan chan []byte
	flushAudioCh chan struct{}
}

type RTPConfig struct {
	LocalIP     string
	LocalPort   int
	PayloadType uint8
	ClockRate   uint32
	Logger      commons.Logger
}

func (c *RTPConfig) Validate() error {
	return c.toCore().Validate()
}

func (c *RTPConfig) toCore() *internal_core.RTPConfig {
	if c == nil {
		return nil
	}
	return &internal_core.RTPConfig{
		LocalIP:     c.LocalIP,
		LocalPort:   c.LocalPort,
		PayloadType: c.PayloadType,
		ClockRate:   c.ClockRate,
		Logger:      c.Logger,
	}
}

func NewRTPHandler(ctx context.Context, config *RTPConfig) (*RTPHandler, error) {
	inner, err := internal_core.NewRTPHandler(ctx, config.toCore())
	if err != nil {
		return nil, err
	}
	return wrapRTPHandler(inner), nil
}

func wrapRTPHandler(inner *internal_core.RTPHandler) *RTPHandler {
	if inner == nil {
		return nil
	}
	return &RTPHandler{inner: inner}
}

func (h *RTPHandler) unwrap() *internal_core.RTPHandler {
	if h == nil {
		return nil
	}
	return h.inner
}

func (h *RTPHandler) Start() {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.Start()
	}
}

func (h *RTPHandler) Stop() error {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.Stop()
	}
	return nil
}

func (h *RTPHandler) IsRunning() bool {
	if h == nil {
		return false
	}
	if h.inner != nil {
		return h.inner.IsRunning()
	}
	return true
}

func (h *RTPHandler) SetRemoteAddr(ip string, port int) {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.SetRemoteAddr(ip, port)
	}
}

func (h *RTPHandler) GetRemoteAddr() *net.UDPAddr {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.GetRemoteAddr()
	}
	return nil
}

func (h *RTPHandler) LocalAddr() (string, int) {
	if h == nil {
		return "", 0
	}
	if h.inner != nil {
		return h.inner.LocalAddr()
	}
	return "", 0
}

func (h *RTPHandler) AudioIn() <-chan []byte {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.AudioIn()
	}
	if h.audioInChan == nil {
		h.audioInChan = make(chan []byte, 100)
	}
	return h.audioInChan
}

func (h *RTPHandler) AudioOut() chan<- []byte {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.AudioOut()
	}
	if h.audioOutChan == nil {
		h.audioOutChan = make(chan []byte, 100)
	}
	return h.audioOutChan
}

func (h *RTPHandler) FlushAudioOut() {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.FlushAudioOut()
		return
	}
	if h.flushAudioCh == nil {
		h.flushAudioCh = make(chan struct{}, 1)
	}
	select {
	case h.flushAudioCh <- struct{}{}:
	default:
	}
}

func (h *RTPHandler) GetCodec() *Codec {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return codecFromCore(h.inner.GetCodec())
	}
	return h.codec
}

func (h *RTPHandler) SetCodec(codec *Codec) {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.SetCodec(codec.toCore())
		return
	}
	h.codec = codec
}

func (h *RTPHandler) GetStats() (sent, received uint64) {
	if h == nil {
		return 0, 0
	}
	if h.inner != nil {
		return h.inner.GetStats()
	}
	return 0, 0
}

func (h *RTPHandler) GetDetailedStats() RTPStats {
	if h == nil {
		return RTPStats{}
	}
	if h.inner != nil {
		return rtpStatsFromCore(h.inner.GetDetailedStats())
	}
	return RTPStats{}
}

type RTPAllocator interface {
	Allocate() (int, error)
	Release(port int)
	InUse() (int, error)
	ReleaseAll(ctx context.Context)
}

type RTPPortAllocator struct {
	inner *internal_core.RTPPortAllocator
}

func NewRTPPortAllocator(client *redis.Client, logger commons.Logger, portStart, portEnd int) *RTPPortAllocator {
	return &RTPPortAllocator{inner: internal_core.NewRTPPortAllocator(client, logger, portStart, portEnd)}
}

func (a *RTPPortAllocator) Init(ctx context.Context) error {
	return a.inner.Init(ctx)
}

func (a *RTPPortAllocator) Allocate() (int, error) {
	return a.inner.Allocate()
}

func (a *RTPPortAllocator) Release(port int) {
	a.inner.Release(port)
}

func (a *RTPPortAllocator) InUse() (int, error) {
	return a.inner.InUse()
}

func (a *RTPPortAllocator) ReleaseAll(ctx context.Context) {
	a.inner.ReleaseAll(ctx)
}
