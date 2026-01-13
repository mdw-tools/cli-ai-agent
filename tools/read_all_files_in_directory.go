package tools

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type ReadAllFilesInDirectoryTool struct {
}

func (this *ReadAllFilesInDirectoryTool) Name() string {
	return "read_all_files_in_directory_tree"
}

func (this *ReadAllFilesInDirectoryTool) Description() string {
	return "Given a path to a folder, recursively read all text (code) files."
}

func (this *ReadAllFilesInDirectoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the directory with files to read (recursively).",
			},
		},
		"required": []string{"path"},
	}
}

func (this *ReadAllFilesInDirectoryTool) Execute(params map[string]interface{}) (string, error) {
	root, ok := params["path"].(string)
	if !ok || root == "" {
		return "", fmt.Errorf("path parameter must be a non-empty string")
	}
	var result strings.Builder
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && len(info.Name()) > 1 {
				return filepath.SkipDir
			}
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		_, _ = fmt.Fprintf(&result, "\n\nFile at: %s\n\n", path)
		reader := io.LimitReader(file, 1024*64)
		content, _ := io.ReadAll(reader)
		if utf8.Valid(content) {
			_, _ = result.Write(content)
		}
		return nil
	})
	return result.String(), err
}

func (this *ReadAllFilesInDirectoryTool) RequiresPermission() bool {
	return false
}
