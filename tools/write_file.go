package tools

import (
	"errors"
	"os"
)

// WriteFileTool implements file writing
type WriteFileTool struct{}

func (this *WriteFileTool) Name() string { return "write_file" }
func (this *WriteFileTool) Description() string {
	return "Write a file. If the file already exists, it will be overwritten."
}
func (this *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write.",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file.",
			},
		},
		"required": []string{"path"},
	}
}
func (this *WriteFileTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", errors.New("path parameter must be a string")
	}
	replace, ok := params["content"].(string)
	if !ok {
		return "", errors.New("content parameter must be a string")
	}
	return replace, os.WriteFile(path, []byte(replace), 0644)
}
func (this *WriteFileTool) RequiresPermission() bool { return true }
