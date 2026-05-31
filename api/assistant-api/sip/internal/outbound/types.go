// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package outbound

import "fmt"

const SIPUserAgent = "RapidaVoiceAI"

type Transport string

const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
	TransportTLS Transport = "tls"
)

type Config struct {
	Address   string
	Port      int
	Transport Transport
	Domain    string
	Headers   map[string]string
}

type Identity struct {
	ToUser   string
	FromUser string
}

type InviteRequest struct {
	Config   Config
	Identity Identity
}

type ContactConfig struct {
	ExternalIP string
	Port       int
	Transport  Transport
}

var ErrFromUserRequired = fmt.Errorf("outbound from user is required")
