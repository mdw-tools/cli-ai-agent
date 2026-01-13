package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mdw-tools/cli-ai-agent/pretty"
	"github.com/mdw-tools/cli-ai-agent/tools"
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
	agent.RegisterTool(&tools.ReadFileTool{})
	agent.RegisterTool(&tools.WriteFileTool{})
	agent.RegisterTool(&tools.ModifyFileTool{})
	agent.RegisterTool(&tools.ReadAllFilesInDirectoryTool{})
	agent.RegisterTool(&tools.RunCommandTool{})
	agent.RegisterTool(&tools.ExecutePythonTool{})

	for {
		fmt.Println(strings.Repeat("#", 80))

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

// Tool interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}) (string, error)
	RequiresPermission() bool
}

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
			Type: "function",
			Function: ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return results
}

func (this *Agent) askPermission(toolName string, params map[string]interface{}) bool {
	fmt.Println(strings.Repeat("#", 80))
	fmt.Printf("\n⚠️  The AI wants to execute: %s\n", toolName)
	fmt.Println("Parameters:")
	for k, v := range params {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Print("Allow? (Y/n): ")
	response := strings.TrimSpace(strings.ToLower(readInput()))
	return response == "" || response == "y" || response == "yes"
}

func (this *Agent) ProcessMessage(userMessage string) error {
	this.conversation = append(this.conversation, Message{
		Role:    "user",
		Content: userMessage,
	})

	// Agentic loop: continue making requests as long as tools are being called
	maxIterations := 10
	for iteration := 0; iteration < maxIterations; iteration++ {
		shouldContinue, err := this.processOneResponse()
		if err != nil {
			return err
		}
		if !shouldContinue {
			break
		}
		fmt.Printf("\n[Continuing agentic loop, iteration %d/%d]\n", iteration+2, maxIterations)
	}
	return nil
}

func (this *Agent) processOneResponse() (shouldContinue bool, err error) {
	// Start spinner while waiting for response
	spinner := pretty.NewSpinner("Waiting for response...")
	spinner.Start()
	defer spinner.Stop()

	req := OllamaRequest{
		Model:    this.model,
		Messages: this.conversation,
		Stream:   true,
		Tools:    this.getToolDefinitions(),
	}

	jsonData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return false, err
	}

	// TODO: implement retry
	request, err := http.NewRequest("POST", this.ollamaURL+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/json")

	// requestDump, err := httputil.DumpRequestOut(request, false)
	// if err != nil {
	// 	return false, err
	// }
	// fmt.Println(strings.Repeat("#", 80))
	// log.Printf("Request dump:\n%s\n\n%s", requestDump, jsonData)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false, err
	}
	defer func() { _ = response.Body.Close() }()

	// Handle streaming response
	scanner := bufio.NewScanner(response.Body)
	var finalMessage Message
	var thinkingDisplayed bool
	var contentDisplayed bool

	for scanner.Scan() {
		spinner.Stop()

		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk OllamaResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			log.Printf("Error parsing chunk: %v\n", err)
			continue
		}

		// Display thinking if present
		if chunk.Message.Thinking != "" {
			if !thinkingDisplayed {
				fmt.Print("\n💭 Thinking: ")
				thinkingDisplayed = true
			}
			fmt.Print(chunk.Message.Thinking)
			finalMessage.Thinking += chunk.Message.Thinking
		}

		// Display content if present
		if chunk.Message.Content != "" {
			if !contentDisplayed {
				if thinkingDisplayed {
					fmt.Println() // New line after thinking
				}
				fmt.Print("\n🤖 Assistant: ")
				contentDisplayed = true
			}
			fmt.Print(chunk.Message.Content)
			finalMessage.Content += chunk.Message.Content
		}

		// Accumulate other fields
		if chunk.Message.Role != "" {
			finalMessage.Role = chunk.Message.Role
		}
		if len(chunk.Message.ToolCalls) > 0 {
			finalMessage.ToolCalls = chunk.Message.ToolCalls
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		spinner.Stop()
		return false, fmt.Errorf("error reading stream: %v", err)
	}

	fmt.Println() // New line after output
	fmt.Println(strings.Repeat("#", 80))

	this.conversation = append(this.conversation, finalMessage)

	// Track tool execution for agentic loop
	var toolsExecuted int
	var anyToolRequiredPermission bool

	for _, toolCall := range finalMessage.ToolCalls {
		toolName := toolCall.Function.Name
		tool, exists := this.tools[toolName]
		if !exists {
			log.Println("🤖 response refers to unknown tool:", toolName)
			continue
		}

		// Check if permission is required
		if tool.RequiresPermission() {
			anyToolRequiredPermission = true
			if !this.askPermission(toolName, toolCall.Function.Arguments) {
				this.conversation = append(this.conversation, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Permission denied for %s", toolName),
				})
				continue
			}
		}

		fmt.Println(strings.Repeat("#", 80))
		fmt.Printf("🔧 Executing tool: %s\n", toolName)
		result, err := tool.Execute(toolCall.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("Error: %v", err)
		}
		fmt.Println(strings.Repeat("#", 80))
		fmt.Println("## Result of tool call:", toolName)
		fmt.Println()
		fmt.Println(result)
		fmt.Println()
		fmt.Println(strings.Repeat("#", 80))

		this.conversation = append(this.conversation, Message{
			Role:    "tool",
			Content: result,
		})
		toolsExecuted++
	}

	// Continue agentic loop if tools were executed and none required permission
	shouldContinue = toolsExecuted > 0 && !anyToolRequiredPermission
	return shouldContinue, nil
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
	Model    string     `json:"model,omitempty"`
	Stream   bool       `json:"stream"` // TODO: rework to utilize streaming (and visualize 'thinking' vs 'content'
	Tools    []ToolCall `json:"tools,omitempty"`
	Messages []Message  `json:"messages,omitempty"`
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
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function,omitempty"`
}
type ToolFunction struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
}
