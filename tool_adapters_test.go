package main

import (
	"encoding/json"
	"testing"
)

func TestClaudeCodeAdapter(t *testing.T) {
	adapter := ClaudeCodeAdapter{}
	tr := &ToolResponse{FilePath: "/tmp/a.md", OldString: "old", NewString: "new"}
	update := UpdateContent{
		Meta: &MetaContent{ClaudeCode: &ClaudeCodeMeta{ToolName: "Edit", ToolResponse: tr}},
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
	adapter := OpenCodeAdapter{}
	diffs := []DiffBlock{{Type: "diff", Path: "/tmp/hello.md", OldText: "old", NewText: "new"}}
	content, err := json.Marshal(diffs)
	if err != nil {
		t.Fatalf("marshal diffs: %v", err)
	}
	update := UpdateContent{
		Title:    "write",
		ToolKind: "edit",
		Content:  content,
		RawOutput: &ToolRawOutput{Metadata: &ToolOutputMetadata{
			Diff: "@@ -1,1 +1,1 @@\n-old\n+new\n",
			Filediff: &FileDiff{
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
