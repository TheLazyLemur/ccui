package anthropic

// DefaultTools returns the default tool definitions for the Anthropic API
func DefaultTools() []Tool {
	return []Tool{
		readTool(),
		writeTool(),
		editTool(),
		bashTool(),
		globTool(),
		grepTool(),
	}
}

func readTool() Tool {
	return Tool{
		Name:        "Read",
		Description: "Reads a file from the local filesystem. Returns content with line numbers.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"file_path": {
					Type:        "string",
					Description: "The absolute path to the file to read",
				},
				"offset": {
					Type:        "number",
					Description: "The line number to start reading from (1-indexed). Only provide if the file is too large to read at once.",
				},
				"limit": {
					Type:        "number",
					Description: "The number of lines to read. Only provide if the file is too large to read at once.",
				},
			},
			Required: []string{"file_path"},
		},
	}
}

func writeTool() Tool {
	return Tool{
		Name:        "Write",
		Description: "Writes content to a file, creating parent directories as needed. Overwrites existing files.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"file_path": {
					Type:        "string",
					Description: "The absolute path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "The content to write to the file",
				},
			},
			Required: []string{"file_path", "content"},
		},
	}
}

func editTool() Tool {
	return Tool{
		Name:        "Edit",
		Description: "Performs exact string replacements in files. The old_string must be unique in the file unless replace_all is true.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"file_path": {
					Type:        "string",
					Description: "The absolute path to the file to modify",
				},
				"old_string": {
					Type:        "string",
					Description: "The text to replace",
				},
				"new_string": {
					Type:        "string",
					Description: "The text to replace it with",
				},
				"replace_all": {
					Type:        "boolean",
					Description: "Replace all occurrences of old_string (default false)",
					Default:     false,
				},
			},
			Required: []string{"file_path", "old_string", "new_string"},
		},
	}
}

func bashTool() Tool {
	return Tool{
		Name:        "Bash",
		Description: "Executes a bash command with optional timeout. Returns combined stdout and stderr.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"command": {
					Type:        "string",
					Description: "The bash command to execute",
				},
				"timeout": {
					Type:        "number",
					Description: "Optional timeout in milliseconds (default 120000, max 600000)",
				},
			},
			Required: []string{"command"},
		},
	}
}

func globTool() Tool {
	return Tool{
		Name:        "Glob",
		Description: "Finds files matching a glob pattern. Returns matching file paths sorted by modification time (newest first).",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"pattern": {
					Type:        "string",
					Description: "The glob pattern to match files against (e.g., \"**/*.go\", \"src/**/*.ts\")",
				},
				"path": {
					Type:        "string",
					Description: "The directory to search in. Defaults to current working directory.",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

func grepTool() Tool {
	return Tool{
		Name:        "Grep",
		Description: "Searches files for a regex pattern. Supports filtering by glob and different output modes.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"pattern": {
					Type:        "string",
					Description: "The regular expression pattern to search for",
				},
				"path": {
					Type:        "string",
					Description: "File or directory to search in. Defaults to current working directory.",
				},
				"glob": {
					Type:        "string",
					Description: "Glob pattern to filter files (e.g., \"*.js\", \"**/*.tsx\")",
				},
				"output_mode": {
					Type:        "string",
					Description: "Output mode: \"files_with_matches\" (default), \"content\", or \"count\"",
					Enum:        []string{"files_with_matches", "content", "count"},
				},
				"-i": {
					Type:        "boolean",
					Description: "Case insensitive search",
				},
				"-A": {
					Type:        "number",
					Description: "Number of lines to show after each match (requires output_mode: content)",
				},
				"-B": {
					Type:        "number",
					Description: "Number of lines to show before each match (requires output_mode: content)",
				},
				"-C": {
					Type:        "number",
					Description: "Number of lines to show before and after each match (requires output_mode: content)",
				},
				"head_limit": {
					Type:        "number",
					Description: "Limit output to first N entries",
				},
			},
			Required: []string{"pattern"},
		},
	}
}
