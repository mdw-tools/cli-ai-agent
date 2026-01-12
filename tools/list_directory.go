package tools

import (
	"fmt"
	"os"
	"strings"
)

// ListDirectoryTool implements directory listing
type ListDirectoryTool struct{}

func (this *ListDirectoryTool) Name() string { return "list_directory" }
func (this *ListDirectoryTool) Description() string {
	return "List files and directories in a given path"
}
func (this *ListDirectoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the directory to list",
			},
		},
		"required": []string{"path"},
	}
}
func (this *ListDirectoryTool) RequiresPermission() bool { return false }
func (this *ListDirectoryTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path parameter must be a non-empty string")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var result strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("[DIR]  %s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("[FILE] %s (%d bytes)\n", entry.Name(), info.Size()))
		}
	}
	return result.String(), nil
}
