package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var Version = "dev"

type Config struct {
	Model     string
	OllamaURL string
}

func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	var config Config

	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	flags.StringVar(&config.Model, "model", "mistral", "The ollama model to use (must already be pulled/downloaded).")
	flags.StringVar(&config.OllamaURL, "ollama-url", "http://localhost:11434", "The URL of the running ollama instance.")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		_, _ = fmt.Fprintf(flags.Output(), "%s [args ...]\n", filepath.Base(os.Args[0]))
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	log.SetPrefix(fmt.Sprintf("[%s] ", config.Model))
	log.Println("🚀 Agentic AI REPL with Ollama")
	log.Println("Type 'exit' to end the session.")
	log.Println("Type 'clear' to clear conversation history.")
	log.Printf("Config: %#v", config)

	agent := NewAgent(config.Model, config.OllamaURL)
	agent.RegisterTool(&ReadFileTool{})
	agent.RegisterTool(&WriteFileTool{})
	agent.RegisterTool(&ListDirectoryTool{})
	agent.RegisterTool(&ListTreeTool{})
	agent.RegisterTool(&RunCommandTool{})
	agent.RegisterTool(&ExecutePythonTool{})

	for {
		fmt.Print("You: ")
		input := readInput()
		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if input == "clear" {
			agent.conversation = agent.conversation[:0]
			fmt.Println("Conversation history cleared.")
			continue
		}

		if err := agent.ProcessMessage(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		fmt.Println()
	}
}

///////////////////////////////////////////////////////////////////////////////

// Agent manages the conversation and tool execution
type Agent struct {
	model        string
	ollamaURL    string
	tools        map[string]Tool
	conversation []Message
}

func NewAgent(model, ollamaURL string) *Agent {
	return &Agent{
		model:     model,
		ollamaURL: ollamaURL,
		tools:     make(map[string]Tool),
	}
}

func (this *Agent) RegisterTool(tool Tool) {
	this.tools[tool.Name()] = tool
}

func (this *Agent) getToolDefinitions() (results []ToolCall) {
	for _, tool := range this.tools {
		results = append(results, ToolCall{
			Function: ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Arguments:   tool.Parameters(),
			},
		})
	}
	return results
}

func (this *Agent) askPermission(toolName string, params map[string]interface{}) bool {
	fmt.Printf("\n⚠️  The AI wants to execute: %s\n", toolName)
	fmt.Printf("Parameters: %v\n", params)
	fmt.Print("Allow? (yes/no): ")
	response := strings.TrimSpace(strings.ToLower(readInput()))
	return response == "yes" || response == "y"
}

func (this *Agent) ProcessMessage(userMessage string) error {
	this.conversation = append(this.conversation, Message{
		Role:    "user",
		Content: userMessage,
	})

	req := OllamaRequest{
		Model:     this.model,
		Messages:  this.conversation,
		Stream:    false,
		ToolCalls: this.getToolDefinitions(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// TODO: implement retry
	request, err := http.NewRequest("POST", this.ollamaURL+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	requestDump, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		return err
	}
	log.Printf("Request dump:\n%s", requestDump)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	responseDump, err := httputil.DumpResponse(response, true)
	if err != nil {
		return err
	}
	log.Printf("Response dump:\n%s", responseDump)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return err
	}

	fmt.Printf("\n🤖 Assistant: %s\n", ollamaResp.Message.Content)

	this.conversation = append(this.conversation, ollamaResp.Message)

	for _, toolCall := range ollamaResp.Message.ToolCalls {
		toolName := toolCall.Function.Name
		tool, exists := this.tools[toolName]
		if !exists {
			log.Println("🤖 response refers to unknown tool:", toolName)
			continue
		}

		// Check if permission is required
		if tool.RequiresPermission() {
			if !this.askPermission(toolName, toolCall.Function.Arguments) {
				this.conversation = append(this.conversation, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Permission denied for %s", toolName),
				})
				continue
			}
		}

		fmt.Printf("🔧 Executing tool: %s\n", toolName)
		result, err := tool.Execute(toolCall.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("Error: %v", err)
		}

		this.conversation = append(this.conversation, Message{
			Role:    "tool",
			Content: result,
		})
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////

func readInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

// Message represents a chat message
type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Thinking  string     `json:"thinking,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// OllamaRequest represents the request to Ollama API
type OllamaRequest struct {
	Model     string     `json:"model,omitempty"`
	Messages  []Message  `json:"messages,omitempty"`
	Stream    bool       `json:"stream"` // TODO: rework to utilize streaming (and visualize 'thinking' vs 'content'
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// OllamaResponse represents the response from Ollama API
type OllamaResponse struct {
	Model     string  `json:"model,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	Message   Message `json:"message,omitempty"`
	Done      bool    `json:"done,omitempty"`
}

// ToolCall represents a tool call in the message
type ToolCall struct {
	Function ToolFunction `json:"function,omitempty"`
}
type ToolFunction struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////

// Tool interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}) (string, error)
	RequiresPermission() bool
}

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

type WriteFileTool struct {
}

func (this *WriteFileTool) Name() string { return "write_file" }
func (this *WriteFileTool) Description() string {
	return "Write a file, either by overwriting the entire contents or just replacing a portion."
}
func (this *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write.",
			},
			"search": map[string]interface{}{
				"type":        "string",
				"description": "A string to search for and, if found, replace. Implies that the supplied path already exists.",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The text to write; used to replace any content matched by 'search' value, otherwise replaces entire contents of file (if it exists).",
			},
		},
		"required": []string{"path"},
	}
}
func (this *WriteFileTool) Execute(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter must be a string")
	}
	search, ok := params["search"].(string)
	if !ok {
		return "", fmt.Errorf("search parameter must be a string")
	}
	replace, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter must be a string")
	}
	raw, err := os.ReadFile(path)
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	content := string(raw)
	content = strings.ReplaceAll(content, search, replace)
	err = os.WriteFile(path, []byte(content), 0644)
	return content, err
}
func (this *WriteFileTool) RequiresPermission() bool { return true }

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
	if !ok {
		return "", fmt.Errorf("path parameter must be a string")
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
	if !ok {
		return "", fmt.Errorf("path parameter must be a string")
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

// RunCommandTool implements shell command execution
type RunCommandTool struct{}

func (this *RunCommandTool) Name() string { return "run_command" }
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
	if !ok {
		return "", fmt.Errorf("command parameter must be a string")
	}
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v\n%s", err, string(output))
	}
	return string(output), nil
}

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
	if !ok {
		return "", fmt.Errorf("script parameter must be a string")
	}
	cmd := exec.Command("python3", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("python execution failed: %v\n%s", err, string(output))
	}
	return string(output), nil
}
