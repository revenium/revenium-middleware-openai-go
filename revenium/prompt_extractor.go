package revenium

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/openai/openai-go/v3"
)

// truncateUTF8Safe truncates a string to maxBytes while preserving UTF-8 validity.
// It ensures we don't cut in the middle of a multi-byte character.
func truncateUTF8Safe(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// Find the last valid UTF-8 character boundary before maxBytes
	for maxBytes > 0 && maxBytes < len(s) {
		// If this byte is a continuation byte (10xxxxxx), back up
		if (s[maxBytes] & 0xC0) == 0x80 {
			maxBytes--
		} else {
			break
		}
	}

	// Validate the result is valid UTF-8
	result := s[:maxBytes]
	if !utf8.ValidString(result) {
		// If still invalid, use rune iteration as fallback
		runes := []rune(s)
		result = ""
		for _, r := range runes {
			if len(result)+utf8.RuneLen(r) > maxBytes {
				break
			}
			result += string(r)
		}
	}

	return result
}

const (
	// MaxPromptLength is the maximum length for captured prompts/responses
	// Fields exceeding this limit will be truncated
	MaxPromptLength = 50000

	// TruncationMarker is appended to truncated content
	TruncationMarker = "...[TRUNCATED]"
)

// PromptData holds extracted prompt information for metering
type PromptData struct {
	// SystemPrompt contains the system message content (if any)
	SystemPrompt string
	// InputMessages contains JSON-serialized non-system messages
	InputMessages string
	// OutputResponse contains the assistant's response content
	OutputResponse string
	// PromptsTruncated indicates if any field was truncated
	PromptsTruncated bool
}

// ExtractPromptsFromParams extracts system prompt and input messages from OpenAI chat params
func ExtractPromptsFromParams(params openai.ChatCompletionNewParams) PromptData {
	data := PromptData{}

	messages := params.Messages
	if len(messages) == 0 {
		return data
	}

	var systemPrompt string
	var userMessages []map[string]interface{}

	for _, msg := range messages {
		// Extract role and content from the message union type
		role, content := extractMessageContent(msg)

		if role == "system" {
			// Take first system message
			if systemPrompt == "" {
				systemPrompt = content
			}
		} else if role != "" {
			// Collect non-system messages
			userMessages = append(userMessages, map[string]interface{}{
				"role":    role,
				"content": content,
			})
		}
	}

	// Apply truncation to system prompt
	if len(systemPrompt) > MaxPromptLength {
		markerLen := len(TruncationMarker)
		truncateAt := MaxPromptLength - markerLen
		systemPrompt = truncateUTF8Safe(systemPrompt, truncateAt) + TruncationMarker
		data.PromptsTruncated = true
		Debug("System prompt truncated to %d characters", MaxPromptLength)
	}
	data.SystemPrompt = systemPrompt

	// Serialize user messages to JSON
	if len(userMessages) > 0 {
		// First truncate individual message contents if needed
		truncatedMessages := make([]map[string]interface{}, 0, len(userMessages))
		halfLimit := MaxPromptLength / 2
		markerLen := len(TruncationMarker)

		for _, msg := range userMessages {
			truncatedMsg := make(map[string]interface{})
			for k, v := range msg {
				truncatedMsg[k] = v
			}

			if content, ok := msg["content"].(string); ok && len(content) > halfLimit {
				truncateAt := halfLimit - markerLen
				truncatedMsg["content"] = truncateUTF8Safe(content, truncateAt) + TruncationMarker
				data.PromptsTruncated = true
			}
			truncatedMessages = append(truncatedMessages, truncatedMsg)
		}

		jsonBytes, err := json.Marshal(truncatedMessages)
		if err != nil {
			Warn("Failed to serialize input messages to JSON: %v", err)
		} else {
			// Note: Individual messages are already truncated above.
			// We avoid truncating the final JSON to prevent invalid JSON.
			data.InputMessages = string(jsonBytes)
		}
	}

	return data
}

// extractMessageContent extracts role and content from a ChatCompletionMessageParamUnion
func extractMessageContent(msg openai.ChatCompletionMessageParamUnion) (role string, content string) {
	// The OpenAI SDK uses union types - we need to handle each variant
	// Try to extract via JSON marshaling as a reliable approach
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return "", ""
	}

	// Extract role
	if r, ok := parsed["role"].(string); ok {
		role = r
	}

	// Extract content - can be string or array of content parts
	switch c := parsed["content"].(type) {
	case string:
		content = c
	case []interface{}:
		// Multimodal content - serialize as JSON
		contentBytes, err := json.Marshal(c)
		if err == nil {
			content = string(contentBytes)
		}
	}

	return role, content
}

// ExtractResponseContent extracts output response from OpenAI chat completion
func ExtractResponseContent(resp *openai.ChatCompletion, promptsTruncated bool) PromptData {
	data := PromptData{
		PromptsTruncated: promptsTruncated,
	}

	if resp == nil || len(resp.Choices) == 0 {
		return data
	}

	choice := resp.Choices[0]
	content := choice.Message.Content

	if content == "" {
		return data
	}

	// Apply truncation if needed
	if len(content) > MaxPromptLength {
		markerLen := len(TruncationMarker)
		truncateAt := MaxPromptLength - markerLen
		content = truncateUTF8Safe(content, truncateAt) + TruncationMarker
		data.PromptsTruncated = true
		Debug("Output response truncated to %d characters", MaxPromptLength)
	}

	data.OutputResponse = content
	return data
}

// ExtractStreamingResponseContent extracts output from accumulated streaming content
func ExtractStreamingResponseContent(accumulatedContent string, promptsTruncated bool) PromptData {
	data := PromptData{
		PromptsTruncated: promptsTruncated,
	}

	if accumulatedContent == "" {
		return data
	}

	content := accumulatedContent

	// Apply truncation if needed
	if len(content) > MaxPromptLength {
		markerLen := len(TruncationMarker)
		truncateAt := MaxPromptLength - markerLen
		content = truncateUTF8Safe(content, truncateAt) + TruncationMarker
		data.PromptsTruncated = true
		Debug("Streaming output response truncated to %d characters", MaxPromptLength)
	}

	data.OutputResponse = content
	return data
}

// AddPromptDataToPayload adds prompt capture fields to a metering payload
func AddPromptDataToPayload(payload map[string]interface{}, data PromptData) {
	if data.SystemPrompt != "" {
		payload["systemPrompt"] = data.SystemPrompt
	}
	if data.InputMessages != "" {
		payload["inputMessages"] = data.InputMessages
	}
	if data.OutputResponse != "" {
		payload["outputResponse"] = data.OutputResponse
	}
	if data.PromptsTruncated {
		payload["promptsTruncated"] = true
	}
}
