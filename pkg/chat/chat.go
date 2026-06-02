package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Chat struct {
	BaseURL   string
	History   []Message
	Input     *bufio.Scanner
}

func New(port int) *Chat {
	return &Chat{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		History: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
		},
		Input: bufio.NewScanner(os.Stdin),
	}
}

func WaitForReady(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Check log file for clues if health never responded
	logFile := fmt.Sprintf("%s/.gollama/logs/port-%s.log",
		os.Getenv("HOME"), strings.TrimPrefix(baseURL, "http://127.0.0.1:"))
	if data, err := os.ReadFile(logFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for i := len(lines) - 1; i >= 0 && i > len(lines)-10; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" && !strings.Contains(line, "\r") {
				return fmt.Errorf("server not ready after %v\nlast log: %s", timeout, line)
			}
		}
	}
	return fmt.Errorf("server not ready after %v", timeout)
}

func (c *Chat) Run() error {
	fmt.Printf("Chatting with llama-server at %s\n", c.BaseURL)
	fmt.Println("Type /exit or /quit to end the conversation.")
	fmt.Println("Multi-line input: type /open, then /close when done.")

	for {
		fmt.Print("\n>>> ")
		if !c.Input.Scan() {
			break
		}
		line := strings.TrimSpace(c.Input.Text())

		if line == "/exit" || line == "/quit" {
			break
		}
		if line == "" {
			continue
		}

		var input strings.Builder
		input.WriteString(line)

		if line == "/open" {
			for {
				fmt.Print("... ")
				if !c.Input.Scan() {
					break
				}
				l := c.Input.Text()
				if strings.TrimSpace(l) == "/close" {
					break
				}
				input.WriteString("\n")
				input.WriteString(l)
			}
		}

		msg := strings.TrimSpace(input.String())
		if msg == "" {
			continue
		}

		c.History = append(c.History, Message{Role: "user", Content: msg})

		response, err := c.sendStream()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			c.History = c.History[:len(c.History)-1]
			continue
		}

		c.History = append(c.History, Message{Role: "assistant", Content: response})
	}

	return nil
}

func (c *Chat) sendStream() (string, error) {
	body := map[string]interface{}{
		"model":       "default",
		"messages":    c.History,
		"stream":      true,
		"max_tokens":  4096,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := http.Post(c.BaseURL+"/v1/chat/completions",
		"application/json", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	reader := bufio.NewReader(resp.Body)
	var fullResponse strings.Builder
	var inReasoning bool

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fullResponse.String(), fmt.Errorf("reading stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content         string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		if delta.ReasoningContent != "" {
			if !inReasoning {
				fmt.Print("\033[3m") // italic
				inReasoning = true
			}
			fmt.Print(delta.ReasoningContent)
			fullResponse.WriteString(delta.ReasoningContent)
		}

		if delta.Content != "" {
			if inReasoning {
				fmt.Print("\033[23m\n\n") // end italic
				inReasoning = false
			}
			fmt.Print(delta.Content)
			fullResponse.WriteString(delta.Content)
		}

		if chunk.Choices[0].FinishReason != nil {
			if inReasoning {
				fmt.Print("\033[23m") // end italic
			}
			fmt.Println()
		}
	}

	return strings.TrimSpace(fullResponse.String()), nil
}
