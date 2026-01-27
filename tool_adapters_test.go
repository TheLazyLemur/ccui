package main

import (
	"ccui/backend"
	"ccui/backend/acp"
	"encoding/json"
	"testing"
)

func TestClaudeCodeAdapter(t *testing.T) {
	adapter := acp.ClaudeCodeAdapter{}
	tr := &acp.ToolResponse{FilePath: "/tmp/a.md", OldString: "old", NewString: "new"}
	update := acp.UpdateContent{
		Meta: &acp.MetaContent{ClaudeCode: &acp.ClaudeCodeMeta{ToolName: "Edit", ToolResponse: tr}},
	}
	if !adapter.CanHandle(update) {
		t.Fatal("expected adapter to handle claude meta")
	}
	if name := adapter.ToolName(update); name != "Edit" {
		t.Fatalf("expected tool name Edit, got %q", name)
	}
	if got := adapter.ToolResponse(update); got != tr {
		t.Fatal("expected tool response passthrough")
	}
}

func TestOpenCodeAdapterToolResponse(t *testing.T) {
	adapter := acp.OpenCodeAdapter{}
	diffs := []backend.DiffBlock{{Type: "diff", Path: "/tmp/hello.md", OldText: "old", NewText: "new"}}
	content, err := json.Marshal(diffs)
	if err != nil {
		t.Fatalf("marshal diffs: %v", err)
	}
	update := acp.UpdateContent{
		Title:    "write",
		ToolKind: "edit",
		Content:  content,
		RawOutput: &acp.ToolRawOutput{Metadata: &acp.ToolOutputMetadata{
			Diff: "@@ -1,1 +1,1 @@\n-old\n+new\n",
			Filediff: &acp.FileDiff{
				File:   "/tmp/hello.md",
				Before: "old",
				After:  "new",
			},
		}},
	}
	if name := adapter.ToolName(update); name != "Write" {
		t.Fatalf("expected tool name Write, got %q", name)
	}
	tr := adapter.ToolResponse(update)
	if tr == nil {
		t.Fatal("expected tool response")
	}
	if tr.FilePath != "/tmp/hello.md" {
		t.Fatalf("expected file path, got %q", tr.FilePath)
	}
	if tr.Content == "" {
		t.Fatal("expected content populated")
	}
	if len(tr.StructuredPatch) == 0 {
		t.Fatal("expected structured patch")
	}
}
