package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillStore reads skill templates from a directory of .md files
type SkillStore struct {
	dir string
}

// NewSkillStore creates a skill store
func NewSkillStore(dir string) *SkillStore {
	return &SkillStore{dir: dir}
}

// List returns available skill names
func (s *SkillStore) List() []string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return names
}

// Get returns the content of a skill
func (s *SkillStore) Get(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, name+".md"))
	if err != nil {
		return "", fmt.Errorf("skill not found: %s", name)
	}
	return string(data), nil
}

// ResolvePrompt replaces $skill-name at the start of a prompt with skill content.
// The remaining text after the skill reference is appended on a new line.
func (s *SkillStore) ResolvePrompt(prompt string) (string, error) {
	if !strings.HasPrefix(prompt, "$") {
		return prompt, nil
	}

	// extract skill name (first word without $)
	parts := strings.SplitN(prompt, " ", 2)
	skillName := strings.TrimPrefix(parts[0], "$")

	content, err := s.Get(skillName)
	if err != nil {
		return "", fmt.Errorf("resolve skill %s: %w", skillName, err)
	}

	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		return content + "\n" + parts[1], nil
	}
	return content, nil
}
