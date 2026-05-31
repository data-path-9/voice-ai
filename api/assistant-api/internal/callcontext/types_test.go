package internal_callcontext

import "testing"

func TestCallContextToAuthIncludesServiceScope(t *testing.T) {
	callContext := &CallContext{
		AuthToken:      "service-token",
		ProjectID:      22,
		OrganizationID: 33,
	}

	auth := callContext.ToAuth()
	if auth.GetCurrentToken() != "service-token" {
		t.Fatalf("expected auth token service-token, got %q", auth.GetCurrentToken())
	}
	if auth.GetCurrentProjectId() == nil || *auth.GetCurrentProjectId() != 22 {
		t.Fatalf("expected project id 22, got %v", auth.GetCurrentProjectId())
	}
	if auth.GetCurrentOrganizationId() == nil || *auth.GetCurrentOrganizationId() != 33 {
		t.Fatalf("expected organization id 33, got %v", auth.GetCurrentOrganizationId())
	}
}

func TestCallContextToAuthOmitsEmptyScopeIDs(t *testing.T) {
	callContext := &CallContext{AuthToken: "service-token"}

	auth := callContext.ToAuth()
	if auth.GetCurrentToken() != "service-token" {
		t.Fatalf("expected auth token service-token, got %q", auth.GetCurrentToken())
	}
	if auth.GetCurrentProjectId() != nil {
		t.Fatalf("expected nil project id, got %v", auth.GetCurrentProjectId())
	}
	if auth.GetCurrentOrganizationId() != nil {
		t.Fatalf("expected nil organization id, got %v", auth.GetCurrentOrganizationId())
	}
}
