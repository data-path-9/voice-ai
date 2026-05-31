// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"testing"

	"github.com/emiago/sipgo"
	"github.com/stretchr/testify/assert"
)

func TestServerHealthSnapshot_NotRunning(t *testing.T) {
	server := &Server{}
	server.state.Store(int32(ServerStateCreated))

	snapshot := server.HealthSnapshot()

	assert.False(t, snapshot.Ready)
	assert.Equal(t, ServerStateCreated, snapshot.State)
	assert.Equal(t, "server_not_running", snapshot.Reason)
}

func TestServerHealthSnapshot_MissingClient(t *testing.T) {
	server := &Server{}
	server.state.Store(int32(ServerStateRunning))

	snapshot := server.HealthSnapshot()

	assert.False(t, snapshot.Ready)
	assert.Equal(t, "sip_client_unavailable", snapshot.Reason)
}

func TestServerHealthSnapshot_RequiresConfiguredExternalIP(t *testing.T) {
	server := runningHealthTestServer(&ListenConfig{Address: "0.0.0.0", Port: 5060, Transport: TransportUDP})

	snapshot := server.HealthSnapshot()

	assert.False(t, snapshot.Ready)
	assert.Equal(t, "external_ip_not_configured", snapshot.Reason)
}

func TestServerHealthSnapshot_RejectsUnspecifiedExternalIP(t *testing.T) {
	server := runningHealthTestServer(&ListenConfig{Address: "0.0.0.0", ExternalIP: "0.0.0.0", Port: 5060, Transport: TransportUDP})

	snapshot := server.HealthSnapshot()

	assert.False(t, snapshot.Ready)
	assert.Equal(t, "external_ip_unspecified", snapshot.Reason)
}

func TestServerHealthSnapshot_RejectsLoopbackExternalIPByDefault(t *testing.T) {
	server := runningHealthTestServer(&ListenConfig{Address: "127.0.0.1", ExternalIP: "127.0.0.1", Port: 5060, Transport: TransportUDP})

	snapshot := server.HealthSnapshot()

	assert.False(t, snapshot.Ready)
	assert.Equal(t, "external_ip_loopback", snapshot.Reason)
}

func TestServerHealthSnapshot_AllowsLoopbackExternalIPWhenExplicitlyEnabled(t *testing.T) {
	server := runningHealthTestServer(&ListenConfig{
		Address:                 "127.0.0.1",
		ExternalIP:              "127.0.0.1",
		AllowLoopbackExternalIP: true,
		Port:                    5060,
		Transport:               TransportUDP,
	})

	snapshot := server.HealthSnapshot()

	assert.True(t, snapshot.Ready)
	assert.Equal(t, "ready", snapshot.Reason)
}

func runningHealthTestServer(listenConfig *ListenConfig) *Server {
	server := &Server{
		client:            &sipgo.Client{},
		server:            &sipgo.Server{},
		dialogClientCache: &sipgo.DialogClientCache{},
		listenConfig:      listenConfig,
		rtpAllocator:      &testRTPAllocator{},
	}
	server.state.Store(int32(ServerStateRunning))
	return server
}

type testRTPAllocator struct {
	nextPort    int
	releasePort int
	inUse       int
}

func (a *testRTPAllocator) Allocate() (int, error) {
	return a.nextPort, nil
}

func (a *testRTPAllocator) Release(port int) {
	a.releasePort = port
}

func (a *testRTPAllocator) InUse() (int, error) {
	return a.inUse, nil
}

func (a *testRTPAllocator) ReleaseAll(ctx context.Context) {}
