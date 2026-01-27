package main

import (
	"ccui/backend/acp"
	"testing"
)

func TestNormalizeToolName(t *testing.T) {
	// Test via ResolveToolName which uses normalizeToolName internally
	update := acp.UpdateContent{Title: "write"}
	if got := acp.ResolveToolName(nil, update); got != "Write" {
		t.Fatalf("expected Write, got %q", got)
	}
	update = acp.UpdateContent{ToolKind: "edit"}
	if got := acp.ResolveToolName(nil, update); got != "Edit" {
		t.Fatalf("expected Edit, got %q", got)
	}
	update = acp.UpdateContent{Title: "custom"}
	if got := acp.ResolveToolName(nil, update); got != "custom" {
		t.Fatalf("expected custom, got %q", got)
	}
}

// Note: parseUnifiedDiff and buildHunksFromTexts tests moved to backend/acp package
