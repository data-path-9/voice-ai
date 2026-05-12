// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"sort"
	"strings"
)

// Registry holds Namespace bindings keyed by the dotted prefix without the
// trailing dot (e.g. "system", "assistant").
type Registry struct {
	namespaces map[string]Namespace
}

// NewRegistry returns an empty Registry. Use With to populate it, or use
// namespace.NewDefaultRegistry for the standard set of namespaces.
func NewRegistry() *Registry {
	return &Registry{namespaces: map[string]Namespace{}}
}

// With registers (or replaces) a Namespace under prefix.
func (r *Registry) With(prefix string, ns Namespace) *Registry {
	r.namespaces[prefix] = ns
	return r
}

// resolve looks up the Namespace for a dotted key like "assistant.id".
func (r *Registry) resolve(key string) (Namespace, string, bool) {
	dot := strings.IndexByte(key, '.')
	if dot < 0 {
		return nil, "", false
	}
	ns, ok := r.namespaces[key[:dot]]
	if !ok {
		return nil, "", false
	}
	return ns, key[dot+1:], true
}

// each iterates registered namespaces in stable (sorted) prefix order.
func (r *Registry) each(fn func(prefix string, ns Namespace)) {
	keys := make([]string, 0, len(r.namespaces))
	for k := range r.namespaces {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn(k, r.namespaces[k])
	}
}
