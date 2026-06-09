// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rapidaai/pkg/commons"
)

const (
	// Redis key for the set of available RTP ports
	// Uses hash tag {rtp:ports} to ensure all RTP keys hash to the same Redis Cluster slot
	rtpAvailableKey = "{rtp:ports}:available"

	// Redis key prefix for per-instance allocated ports (for crash recovery)
	// Uses hash tag {rtp:ports} to ensure all RTP keys hash to the same Redis Cluster slot
	rtpAllocatedPrefix = "{rtp:ports}:allocated:"

	// TTL for per-instance allocated port tracking (crash recovery)
	rtpAllocatedTTL = 10 * time.Minute
)

// RTPPortAllocator manages distributed allocation of RTP ports via Redis.
// RTP ports are even-numbered per RFC 3550 (RTCP uses the next odd port).
// Thread-safe across multiple server instances via Redis atomic operations.
type RTPAllocator interface {
	Allocate() (int, error)
	Release(port int)
	InUse() (int, error)
	ReleaseAll(ctx context.Context)
}

type RTPPortAllocator struct {
	client     *redis.Client
	logger     commons.Logger
	portStart  int
	portEnd    int
	instanceID string // unique ID for this server instance (crash recovery)
}

// NewRTPPortAllocator creates a Redis-backed distributed port allocator for the given range [start, end).
// Ports are allocated as even numbers per RTP convention.
// The allocator initializes the Redis available-ports set on first use.
func NewRTPPortAllocator(client *redis.Client, logger commons.Logger, portStart, portEnd int) *RTPPortAllocator {
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s:%d", hostname, os.Getpid())

	return &RTPPortAllocator{
		client:     client,
		logger:     logger,
		portStart:  portStart,
		portEnd:    portEnd,
		instanceID: instanceID,
	}
}

// initLuaScript atomically initializes the available ports set in Redis
// only if it doesn't already exist (prevents wiping ports on restart).
var initLuaScript = redis.NewScript(`
	local key = KEYS[1]
	local exists = redis.call('EXISTS', key)
	if exists == 0 then
		for i = 1, #ARGV do
			redis.call('SADD', key, ARGV[i])
		end
		return #ARGV
	end
	return 0
`)

// Init populates the Redis available-ports set with all even ports in the range.
// Safe to call on every startup — only populates if the set doesn't already exist.
func (a *RTPPortAllocator) Init(ctx context.Context) error {
	if a.client == nil {
		return fmt.Errorf("redis connection not available for RTP port allocator")
	}

	// Build list of even-numbered ports
	start := a.portStart
	if start%2 != 0 {
		start++
	}

	ports := make([]interface{}, 0, (a.portEnd-start)/2)
	for port := start; port < a.portEnd; port += 2 {
		ports = append(ports, port)
	}

	if len(ports) == 0 {
		return fmt.Errorf("no valid RTP ports in range %d-%d", a.portStart, a.portEnd)
	}

	// Atomically initialize only if the set doesn't exist
	result, err := initLuaScript.Run(ctx, a.client, []string{rtpAvailableKey}, ports...).Int()
	if err != nil {
		return fmt.Errorf("failed to initialize RTP port pool in Redis: %w", err)
	}

	if result > 0 {
		a.logger.Info("Initialized RTP port pool in Redis",
			"ports_added", result,
			"range_start", a.portStart,
			"range_end", a.portEnd)
	} else {
		a.logger.Debugw("RTP port pool already exists in Redis, skipping initialization")
	}

	// Reclaim any ports from a previous crashed instance with this same ID
	a.reclaimCrashedPorts(ctx)

	return nil
}

// allocateLuaScript atomically pops a port from available and adds it to the instance's allocated set.
var allocateLuaScript = redis.NewScript(`
	local port = redis.call('SPOP', KEYS[1])
	if port == false then
		return -1
	end
	redis.call('SADD', KEYS[2], port)
	return port
`)

// Allocate returns the next available even-numbered port from the distributed pool.
// Returns an error if no ports are available.
func (a *RTPPortAllocator) Allocate() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.client == nil {
		return 0, fmt.Errorf("redis connection not available for RTP port allocation")
	}

	instanceKey := rtpAllocatedPrefix + a.instanceID

	// Atomically pop from available and track in instance set
	result, err := allocateLuaScript.Run(ctx, a.client, []string{rtpAvailableKey, instanceKey}).Int()
	if err != nil {
		return 0, fmt.Errorf("failed to allocate RTP port from Redis: %w", err)
	}

	if result == -1 {
		inUse, _ := a.InUse()
		return 0, fmt.Errorf("no RTP ports available in range %d-%d (%d in use)",
			a.portStart, a.portEnd, inUse)
	}

	// Refresh TTL on instance tracking key
	a.client.Expire(ctx, instanceKey, rtpAllocatedTTL)

	a.logger.Debugw("Allocated RTP port", "port", result, "instance", a.instanceID)
	return result, nil
}

// releaseLuaScript atomically removes from instance set and adds back to available.
var releaseLuaScript = redis.NewScript(`
	redis.call('SREM', KEYS[2], ARGV[1])
	redis.call('SADD', KEYS[1], ARGV[1])
	return 1
`)

// Release returns a port back to the distributed pool.
func (a *RTPPortAllocator) Release(port int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.client == nil {
		a.logger.Error("Redis connection not available for RTP port release", "port", port)
		return
	}

	instanceKey := rtpAllocatedPrefix + a.instanceID

	_, err := releaseLuaScript.Run(ctx, a.client, []string{rtpAvailableKey, instanceKey}, port).Result()
	if err != nil {
		a.logger.Error("Failed to release RTP port to Redis", "port", port, "error", err)
		return
	}

	a.logger.Debugw("Released RTP port", "port", port, "instance", a.instanceID)
}

// InUse returns the number of currently allocated ports (across all instances).
func (a *RTPPortAllocator) InUse() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.client == nil {
		return 0, fmt.Errorf("redis connection not available")
	}

	// Total ports minus available = in use
	start := a.portStart
	if start%2 != 0 {
		start++
	}
	totalPorts := (a.portEnd - start) / 2

	available, err := a.client.SCard(ctx, rtpAvailableKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get available port count: %w", err)
	}

	return totalPorts - int(available), nil
}

// reclaimCrashedPorts moves any ports tracked under this instance's key back to the available pool.
// This handles the case where a previous instance with the same hostname:pid crashed.
func (a *RTPPortAllocator) reclaimCrashedPorts(ctx context.Context) {
	if a.client == nil {
		return
	}

	instanceKey := rtpAllocatedPrefix + a.instanceID

	// Get all ports allocated to this instance (from a previous crash)
	ports, err := a.client.SMembers(ctx, instanceKey).Result()
	if err != nil {
		a.logger.Warn("Failed to check crashed instance ports", "instance", a.instanceID, "error", err)
		return
	}

	if len(ports) == 0 {
		return
	}

	a.logger.Warn("Reclaiming ports from crashed instance",
		"instance", a.instanceID,
		"ports_count", len(ports))

	// Move each port back to available
	for _, portStr := range ports {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		_, err = releaseLuaScript.Run(ctx, a.client, []string{rtpAvailableKey, instanceKey}, port).Result()
		if err != nil {
			a.logger.Warn("Failed to reclaim port", "port", port, "error", err)
		}
	}

	a.logger.Info("Reclaimed crashed instance ports",
		"instance", a.instanceID,
		"ports_reclaimed", len(ports))
}

// ReleaseAll releases all ports allocated by this instance back to the pool.
// Should be called during graceful shutdown.
func (a *RTPPortAllocator) ReleaseAll(ctx context.Context) {
	if a.client == nil {
		return
	}

	instanceKey := rtpAllocatedPrefix + a.instanceID

	ports, err := a.client.SMembers(ctx, instanceKey).Result()
	if err != nil {
		a.logger.Error("Failed to get allocated ports for release", "error", err)
		return
	}

	for _, portStr := range ports {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		a.Release(port)
	}

	// Clean up instance key
	a.client.Del(ctx, instanceKey)

	a.logger.Info("Released all RTP ports on shutdown",
		"instance", a.instanceID,
		"ports_released", len(ports))
}
