package tools

import (
	"fmt"
	"os"
)

// ReadFileTool implements file reading
type ReadFileTool struct{}

func (this *ReadFileTool) Name() string { return "read_file" }
func (this *ReadFileTool) Description() string {
	return "Read the contents of a file"
}
func (this *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}
func (this *ReadFileTool) RequiresPermission() bool { return false }
func (this *ReadFileTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter must be a string")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
