package tools

import (
	"fmt"
	"os/exec"
)

// ExecutePythonTool implements Python script execution
type ExecutePythonTool struct{}

func (this *ExecutePythonTool) Name() string { return "execute_python" }
func (this *ExecutePythonTool) Description() string {
	return "Execute a Python script and return its output"
}
func (this *ExecutePythonTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"script": map[string]interface{}{
				"type":        "string",
				"description": "The Python code to execute",
			},
		},
		"required": []string{"script"},
	}
}
func (this *ExecutePythonTool) RequiresPermission() bool { return true }
func (this *ExecutePythonTool) Execute(params map[string]interface{}) (string, error) {
	script, ok := params["script"].(string)
	if !ok || script == "" {
		return "", fmt.Errorf("script parameter must be a non-empty string")
	}
	cmd := exec.Command("python3", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("python execution failed: %v\n%s", err, string(output))
	}
	return string(output), nil
}
