package revenium

import (
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
)

func TestExtractPromptsFromParams(t *testing.T) {
	tests := []struct {
		name           string
		params         openai.ChatCompletionNewParams
		wantSystem     bool
		wantInput      bool
		wantTruncated  bool
	}{
		{
			name: "empty messages",
			params: openai.ChatCompletionNewParams{
				Model:    "gpt-4",
				Messages: []openai.ChatCompletionMessageParamUnion{},
			},
			wantSystem:    false,
			wantInput:     false,
			wantTruncated: false,
		},
		{
			name: "with system and user messages",
			params: openai.ChatCompletionNewParams{
				Model: "gpt-4",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a helpful assistant."),
					openai.UserMessage("Hello, how are you?"),
				},
			},
			wantSystem:    true,
			wantInput:     true,
			wantTruncated: false,
		},
		{
			name: "user only messages",
			params: openai.ChatCompletionNewParams{
				Model: "gpt-4",
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("What is the weather?"),
				},
			},
			wantSystem:    false,
			wantInput:     true,
			wantTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPromptsFromParams(tt.params)

			hasSystem := result.SystemPrompt != ""
			hasInput := result.InputMessages != ""

			if hasSystem != tt.wantSystem {
				t.Errorf("SystemPrompt: got %v, want %v", hasSystem, tt.wantSystem)
			}
			if hasInput != tt.wantInput {
				t.Errorf("InputMessages: got %v, want %v", hasInput, tt.wantInput)
			}
			if result.PromptsTruncated != tt.wantTruncated {
				t.Errorf("PromptsTruncated: got %v, want %v", result.PromptsTruncated, tt.wantTruncated)
			}
		})
	}
}

func TestExtractResponseContent(t *testing.T) {
	tests := []struct {
		name          string
		resp          *openai.ChatCompletion
		truncated     bool
		wantOutput    bool
		wantTruncated bool
	}{
		{
			name:          "nil response",
			resp:          nil,
			truncated:     false,
			wantOutput:    false,
			wantTruncated: false,
		},
		{
			name: "empty choices",
			resp: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{},
			},
			truncated:     false,
			wantOutput:    false,
			wantTruncated: false,
		},
		{
			name: "with content",
			resp: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Hello! I'm doing great.",
						},
					},
				},
			},
			truncated:     false,
			wantOutput:    true,
			wantTruncated: false,
		},
		{
			name: "preserves input truncation flag",
			resp: &openai.ChatCompletion{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Short response",
						},
					},
				},
			},
			truncated:     true,
			wantOutput:    true,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractResponseContent(tt.resp, tt.truncated)

			hasOutput := result.OutputResponse != ""
			if hasOutput != tt.wantOutput {
				t.Errorf("OutputResponse: got %v, want %v", hasOutput, tt.wantOutput)
			}
			if result.PromptsTruncated != tt.wantTruncated {
				t.Errorf("PromptsTruncated: got %v, want %v", result.PromptsTruncated, tt.wantTruncated)
			}
		})
	}
}

func TestExtractStreamingResponseContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		truncated     bool
		wantOutput    bool
		wantTruncated bool
	}{
		{
			name:          "empty content",
			content:       "",
			truncated:     false,
			wantOutput:    false,
			wantTruncated: false,
		},
		{
			name:          "normal content",
			content:       "This is a streaming response.",
			truncated:     false,
			wantOutput:    true,
			wantTruncated: false,
		},
		{
			name:          "preserves input truncation",
			content:       "Some content",
			truncated:     true,
			wantOutput:    true,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStreamingResponseContent(tt.content, tt.truncated)

			hasOutput := result.OutputResponse != ""
			if hasOutput != tt.wantOutput {
				t.Errorf("OutputResponse: got %v, want %v", hasOutput, tt.wantOutput)
			}
			if result.PromptsTruncated != tt.wantTruncated {
				t.Errorf("PromptsTruncated: got %v, want %v", result.PromptsTruncated, tt.wantTruncated)
			}
		})
	}
}

func TestTruncation(t *testing.T) {
	// Create content longer than MaxPromptLength
	longContent := strings.Repeat("a", MaxPromptLength+1000)

	result := ExtractStreamingResponseContent(longContent, false)

	if !result.PromptsTruncated {
		t.Error("Expected PromptsTruncated to be true for long content")
	}

	if len(result.OutputResponse) != MaxPromptLength {
		t.Errorf("Expected truncated length %d, got %d", MaxPromptLength, len(result.OutputResponse))
	}

	if !strings.HasSuffix(result.OutputResponse, TruncationMarker) {
		t.Error("Expected truncated content to end with truncation marker")
	}
}

func TestAddPromptDataToPayload(t *testing.T) {
	payload := make(map[string]interface{})

	data := PromptData{
		SystemPrompt:     "You are helpful",
		InputMessages:    `[{"role":"user","content":"Hi"}]`,
		OutputResponse:   "Hello there!",
		PromptsTruncated: true,
	}

	AddPromptDataToPayload(payload, data)

	if payload["systemPrompt"] != "You are helpful" {
		t.Error("systemPrompt not added to payload")
	}
	if payload["inputMessages"] != `[{"role":"user","content":"Hi"}]` {
		t.Error("inputMessages not added to payload")
	}
	if payload["outputResponse"] != "Hello there!" {
		t.Error("outputResponse not added to payload")
	}
	if payload["promptsTruncated"] != true {
		t.Error("promptsTruncated not added to payload")
	}
}

func TestAddPromptDataToPayload_EmptyFields(t *testing.T) {
	payload := make(map[string]interface{})

	data := PromptData{
		SystemPrompt:     "",
		InputMessages:    "",
		OutputResponse:   "",
		PromptsTruncated: false,
	}

	AddPromptDataToPayload(payload, data)

	// Empty fields should not be added
	if _, ok := payload["systemPrompt"]; ok {
		t.Error("empty systemPrompt should not be added to payload")
	}
	if _, ok := payload["inputMessages"]; ok {
		t.Error("empty inputMessages should not be added to payload")
	}
	if _, ok := payload["outputResponse"]; ok {
		t.Error("empty outputResponse should not be added to payload")
	}
	if _, ok := payload["promptsTruncated"]; ok {
		t.Error("false promptsTruncated should not be added to payload")
	}
}
