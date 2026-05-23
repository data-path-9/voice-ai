// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package namespace

import (
	"github.com/rapidaai/api/assistant-api/internal/variable"
)

// NewDefaultRegistry returns a Registry with all globally-available
// namespaces registered: system, assistant, conversation, session, argument,
// metadata, option, client, analysis.
func NewDefaultRegistry() *variable.Registry {
	r := variable.NewRegistry()
	r.With("system", &SystemNamespace{})
	r.With("assistant", &AssistantNamespace{})
	r.With("conversation", &ConversationNamespace{})
	r.With("session", &SessionNamespace{})
	r.With("argument", &ArgumentNamespace{})
	r.With("metadata", &MetadataNamespace{})
	r.With("option", &OptionNamespace{})
	r.With("client", &MetadataPrefixNamespace{Prefix: "client."})
	r.With("analysis", &MetadataPrefixNamespace{Prefix: "analysis."})
	return r
}
