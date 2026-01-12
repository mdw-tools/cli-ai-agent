package tools

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ModifyFileTool implements file modifications
type ModifyFileTool struct{}

func (this *ModifyFileTool) Name() string { return "modify_file" }
func (this *ModifyFileTool) Description() string {
	return "Modify a file by replacing the portion provided."
}
func (this *ModifyFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write (must already exist).",
			},
			"search": map[string]interface{}{
				"type":        "string",
				"description": "A search text.",
			},
			"replace": map[string]interface{}{
				"type":        "string",
				"description": "The replacement text.",
			},
		},
		"required": []string{"path"},
	}
}
func (this *ModifyFileTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", errors.New("path parameter must be a string")
	}
	search, ok := params["search"].(string)
	if !ok || search == "" {
		return "", errors.New("search parameter must be a non-empty string")
	}
	replace, ok := params["replace"].(string)
	if !ok {
		return "", errors.New("replace parameter must be a string")
	}
	fmt.Println("reading file:", path)
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	fmt.Println("Contains search?", strings.Contains(string(raw), search))
	content := strings.ReplaceAll(string(raw), search, replace)
	fmt.Println("writing file:", path)
	err = os.WriteFile(path, []byte(content), 0644)
	fmt.Println("Length of old:", len(string(raw)))
	fmt.Println("Length of new:", len(content))
	return content, err
}
func (this *ModifyFileTool) RequiresPermission() bool { return true }
