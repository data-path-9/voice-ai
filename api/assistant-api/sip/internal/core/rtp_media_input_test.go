// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPHandler_ProcessInboundRTPPacketDropsNonAudioPayload(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMU
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU)

	audioPayloads, acceptedAudio := handler.processInboundRTPPacket(&RTPPacket{
		PayloadType:    101,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0x01},
	})

	assert.False(t, acceptedAudio)
	assert.Empty(t, audioPayloads)
	assert.Equal(t, uint64(1), handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_ProcessInboundRTPPacketAcceptsNegotiatedAudioPayload(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMA
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMA)

	audioPayloads, acceptedAudio := handler.processInboundRTPPacket(&RTPPacket{
		PayloadType:    CodecPCMA.PayloadType,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0xD5},
	})

	require.True(t, acceptedAudio)
	assert.Equal(t, [][]byte{{0xD5}}, audioPayloads)
	assert.Zero(t, handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_WriteInboundAudioPayloadsCountsInputQueueDrops(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMU
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU)
	handler.audioInChan = make(chan []byte, 1)
	handler.audioInChan <- []byte{0x01}

	stopped := handler.writeInboundAudioPayloads([][]byte{{0x02}}, 2)

	assert.False(t, stopped)
	assert.Equal(t, uint64(1), handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_StopOwnsLoopShutdownBeforeClosingChannels(t *testing.T) {
	handler := newTestRTPHandler()
	handler.loops.Add(1)
	loopStarted := make(chan struct{})
	go func() {
		defer handler.loops.Done()
		close(loopStarted)
		<-handler.ctx.Done()
		_ = handler.writeInboundAudioPayloads([][]byte{{0xFF}}, 1)
	}()

	<-loopStarted
	require.NoError(t, handler.Stop())
	require.NoError(t, handler.Stop())

	require.Eventually(t, func() bool {
		select {
		case _, ok := <-handler.audioInChan:
			return !ok
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		select {
		case _, ok := <-handler.audioOutChan:
			return !ok
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}
