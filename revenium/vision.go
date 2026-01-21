package revenium

import (
	"github.com/openai/openai-go/v3"
)

// HasVisionContent checks if any message in the chat completion request contains image content.
// This is used for GPT-4 Vision requests where users send images for analysis.
// Returns true if any message contains image_url content parts.
func HasVisionContent(params openai.ChatCompletionNewParams) bool {
	// Check each message for image content
	for _, msg := range params.Messages {
		if hasImageInMessage(msg) {
			return true
		}
	}
	return false
}

// hasImageInMessage checks if a single message contains image content.
// The OpenAI SDK uses a union struct pattern with pointer fields.
func hasImageInMessage(msg openai.ChatCompletionMessageParamUnion) bool {
	// Check if this is a user message (only user messages can contain image content)
	if msg.OfUser == nil {
		return false
	}

	// Check the user message content for image parts
	return hasImageInUserMessageContent(msg.OfUser.Content)
}

// hasImageInUserMessageContent checks if user message content contains image parts
func hasImageInUserMessageContent(content openai.ChatCompletionUserMessageParamContentUnion) bool {
	// Check if content is an array of parts (multi-part content)
	// When it's a simple string, OfArrayOfContentParts will be nil
	if content.OfArrayOfContentParts == nil {
		return false
	}

	// Check each content part for image content
	for _, part := range content.OfArrayOfContentParts {
		if isImagePart(part) {
			return true
		}
	}

	return false
}

// isImagePart checks if a content part is an image
func isImagePart(part openai.ChatCompletionContentPartUnionParam) bool {
	// Check if the OfImageURL field is set (non-nil means it's an image part)
	return part.OfImageURL != nil
}
