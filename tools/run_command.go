package tools

import (
	"fmt"
	"os/exec"
)

// RunCommandTool implements shell command execution
type RunCommandTool struct{}

func (this *RunCommandTool) Name() string { return "run_shell_command" }
func (this *RunCommandTool) Description() string {
	return "Execute a shell command and return its output"
}
func (this *RunCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}
func (this *RunCommandTool) RequiresPermission() bool { return true }
func (this *RunCommandTool) Execute(params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("command parameter must be a non-empty string")
	}
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v\n%s", err, string(output))
	}
	return string(output), nil
}
