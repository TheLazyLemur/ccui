package acp

import (
	"testing"
)

func TestParseUnifiedDiff(t *testing.T) {
	diffText := "Index: /tmp/hello.md\n" +
		"===================================================================\n" +
		"--- /tmp/hello.md\n" +
		"+++ /tmp/hello.md\n" +
		"@@ -1,3 +1,5 @@\n" +
		" # Hello\n" +
		" \n" +
		"-This is a simple hello markdown file.\n" +
		"+This is a simple hello markdown file.\n" +
		"+\n" +
		"+Created by: Dan\n" +
		"\\ No newline at end of file\n"
	hunks := parseUnifiedDiff(diffText)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	hunk := hunks[0]
	if hunk.OldStart != 1 || hunk.OldLines != 3 || hunk.NewStart != 1 || hunk.NewLines != 5 {
		t.Fatalf("unexpected hunk header: %+v", hunk)
	}
	if len(hunk.Lines) == 0 || hunk.Lines[0] != " # Hello" {
		t.Fatalf("unexpected hunk lines: %+v", hunk.Lines)
	}
}

func TestBuildHunksFromTexts(t *testing.T) {
	hunks := buildHunksFromTexts("a\nb", "a\nb\nc")
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	if hunks[0].OldLines != 2 || hunks[0].NewLines != 3 {
		t.Fatalf("unexpected hunk sizes: %+v", hunks[0])
	}
	expected := []string{"-a", "-b", "+a", "+b", "+c"}
	if len(hunks[0].Lines) != len(expected) {
		t.Fatalf("unexpected lines length: %d", len(hunks[0].Lines))
	}
	for i, line := range expected {
		if hunks[0].Lines[i] != line {
			t.Fatalf("line %d mismatch: %q", i, hunks[0].Lines[i])
		}
	}
}

func TestNormalizeToolName(t *testing.T) {
	if got := normalizeToolName("write", ""); got != "Write" {
		t.Fatalf("expected Write, got %q", got)
	}
	if got := normalizeToolName("", "edit"); got != "Edit" {
		t.Fatalf("expected Edit, got %q", got)
	}
	if got := normalizeToolName("custom", ""); got != "custom" {
		t.Fatalf("expected custom, got %q", got)
	}
}
