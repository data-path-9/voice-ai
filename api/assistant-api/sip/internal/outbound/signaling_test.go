// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package outbound

import (
	"testing"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInviteHeaders_Deterministic(t *testing.T) {
	request := testInviteRequest()
	request.Config.Headers = map[string]string{
		"X-Zeta":              "last",
		"X-Alpha":             "first",
		"Allow":               "bad-override",
		"P-Asserted-Identity": "bad-override",
	}

	headers, err := BuildInviteHeaders(request)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"From",
		"P-Asserted-Identity",
		"Allow",
		"User-Agent",
		"X-Alpha",
		"X-Zeta",
	}, headerNames(headers))
	assert.Equal(t, outboundAllowHeaderValue, headers[2].Value())
	assert.Equal(t, SIPUserAgent, headers[3].Value())
	assert.Equal(t, "first", headers[4].Value())
	assert.Equal(t, "last", headers[5].Value())
}

func TestBuildInviteHeaders_AllowsRouteHeader(t *testing.T) {
	request := testInviteRequest()
	request.Config.Headers = map[string]string{
		"Route": "<sip:proxy.example.com;lr>",
	}

	headers, err := BuildInviteHeaders(request)
	require.NoError(t, err)

	require.Len(t, headers, 5)
	assert.Equal(t, "Route", headers[4].Name())
	assert.Equal(t, "<sip:proxy.example.com;lr>", headers[4].Value())
}

func TestBuildContactHeader_Transport(t *testing.T) {
	cases := []struct {
		name           string
		transport      Transport
		expectedScheme string
		expectedParam  string
	}{
		{name: "udp", transport: TransportUDP, expectedScheme: "sip"},
		{name: "tcp", transport: TransportTCP, expectedScheme: "sip", expectedParam: "tcp"},
		{name: "tls", transport: TransportTLS, expectedScheme: "sips", expectedParam: "tls"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contact := BuildContactHeader(ContactConfig{
				ExternalIP: "203.0.113.10",
				Port:       5061,
				Transport:  tc.transport,
			})

			assert.Equal(t, tc.expectedScheme, contact.Address.Scheme)
			assert.Equal(t, "203.0.113.10", contact.Address.Host)
			assert.Equal(t, 5061, contact.Address.Port)
			if tc.expectedParam == "" {
				assert.Nil(t, contact.Address.UriParams)
				return
			}
			transport, ok := contact.Address.UriParams.Get("transport")
			require.True(t, ok)
			assert.Equal(t, tc.expectedParam, transport)
		})
	}
}

func TestNormalizeDialogRouteSet_UsesRecordRoute(t *testing.T) {
	inviteRequest := sip.NewRequest(sip.INVITE, sip.Uri{Scheme: "sip", User: "callee", Host: "trunk.example.com"})
	inviteRequest.AppendHeader(sip.NewHeader("Route", "<sip:initial.example.com;lr>"))
	inviteResponse := sip.NewResponseFromRequest(inviteRequest, 200, "OK", nil)
	inviteResponse.AppendHeader(sip.NewHeader("Contact", "<sip:uas@carrier.example.com>"))
	inviteResponse.AppendHeader(sip.NewHeader("Record-Route", "<sip:p2.example.com;lr>"))
	inviteResponse.AppendHeader(sip.NewHeader("Record-Route", "<sip:p1.example.com;lr>"))
	dialogSession := testDialogClientSession(inviteRequest, inviteResponse)

	NormalizeDialogRouteSet(dialogSession)

	assert.Empty(t, inviteRequest.GetHeaders("Route"))
}

func testInviteRequest() InviteRequest {
	return InviteRequest{
		Config: Config{
			Address:   "trunk.example.com",
			Port:      5060,
			Transport: TransportUDP,
			Domain:    "example.com",
		},
		Identity: Identity{
			ToUser:   "+15551234567",
			FromUser: "+15557654321",
		},
	}
}

func headerNames(headers []sip.Header) []string {
	names := make([]string, 0, len(headers))
	for _, header := range headers {
		names = append(names, header.Name())
	}
	return names
}

func testDialogClientSession(inviteRequest *sip.Request, inviteResponse *sip.Response) *sipgo.DialogClientSession {
	return &sipgo.DialogClientSession{
		Dialog: sipgo.Dialog{
			InviteRequest:  inviteRequest,
			InviteResponse: inviteResponse,
		},
	}
}
