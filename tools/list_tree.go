package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListTreeTool implements recursive directory tree listing
type ListTreeTool struct{}

func (this *ListTreeTool) Name() string { return "list_tree" }
func (this *ListTreeTool) Description() string {
	return "List all files and directories recursively in a tree structure"
}
func (this *ListTreeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Root path to list from",
			},
			"max_depth": map[string]interface{}{
				"type":        "number",
				"description": "Maximum depth to traverse (optional, default 5)",
			},
		},
		"required": []string{"path"},
	}
}
func (this *ListTreeTool) RequiresPermission() bool { return false }
func (this *ListTreeTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path parameter must be a non-empty string")
	}
	maxDepth := 5
	if d, ok := params["max_depth"].(float64); ok {
		maxDepth = int(d)
	}
	var result strings.Builder
	err := this.walkTree(path, "", 0, maxDepth, &result)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
func (this *ListTreeTool) walkTree(path, prefix string, depth, maxDepth int, result *strings.Builder) error {
	if depth > maxDepth {
		return nil
	}
	base := filepath.Base(path)
	if base == ".git" || base == ".idea" || base == ".claude" {
		return nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for i, entry := range entries {
		isLast := i == len(entries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, entry.Name()))
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			err = this.walkTree(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, result)
			if err != nil {
				return err
			}
		} else {
			result.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, entry.Name()))
		}
	}
	return nil
}
