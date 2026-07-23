package model

import "testing"

func TestAPIKeyModelAllowedAllowList(t *testing.T) {
	k := &APIKey{SupportedModels: "gpt-4o,claude-3", ModelListMode: "allow"}
	if !k.ModelAllowed("gpt-4o") || k.ModelAllowed("gpt-4.1") {
		t.Fatalf("allow list failed")
	}
	empty := &APIKey{}
	if !empty.ModelAllowed("anything") {
		t.Fatalf("empty allow list should allow all")
	}
}

func TestAPIKeyModelAllowedDenyList(t *testing.T) {
	k := &APIKey{SupportedModels: "gpt-4o", ModelListMode: "deny"}
	if k.ModelAllowed("gpt-4o") || !k.ModelAllowed("claude-3") {
		t.Fatalf("deny list failed")
	}
	emptyDeny := &APIKey{ModelListMode: "deny"}
	if !emptyDeny.ModelAllowed("gpt-4o") {
		t.Fatalf("empty deny list should allow all")
	}
}
