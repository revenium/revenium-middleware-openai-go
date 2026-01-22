package revenium

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Images returns the images interface for generating images with metering
func (r *ReveniumOpenAI) Images() *ImagesInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &ImagesInterface{
		client:   r.client,
		config:   r.config,
		provider: r.provider,
		parent:   r,
	}
}

// ImagesInterface provides methods for image generation with metering
type ImagesInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI
}

// Generate creates images using DALL-E with automatic metering
func (i *ImagesInterface) Generate(ctx context.Context, params openai.ImageGenerateParams, opts ...option.RequestOption) (*openai.ImagesResponse, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Get requested image count from params (default is 1)
	requestedCount := 1
	if params.N.Value != 0 {
		requestedCount = int(params.N.Value)
	}

	// Call OpenAI Images API
	resp, err := i.client.Images.Generate(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		i.parent.wg.Add(1)
		go func() {
			defer i.parent.wg.Done()
			i.sendImageMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error(), requestedCount)
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	i.parent.wg.Add(1)
	go func() {
		defer i.parent.wg.Done()
		i.sendImageMeteringData(ctx, resp, string(params.Model), metadata, duration, requestTime, requestedCount)
	}()

	return resp, nil
}

// Edit creates image edits with automatic metering
func (i *ImagesInterface) Edit(ctx context.Context, params openai.ImageEditParams, opts ...option.RequestOption) (*openai.ImagesResponse, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Get requested image count from params (default is 1)
	requestedCount := 1
	if params.N.Value != 0 {
		requestedCount = int(params.N.Value)
	}

	// Call OpenAI Images Edit API
	resp, err := i.client.Images.Edit(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		i.parent.wg.Add(1)
		go func() {
			defer i.parent.wg.Done()
			i.sendImageMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error(), requestedCount)
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	i.parent.wg.Add(1)
	go func() {
		defer i.parent.wg.Done()
		i.sendImageMeteringData(ctx, resp, string(params.Model), metadata, duration, requestTime, requestedCount)
	}()

	return resp, nil
}

// NewVariation creates image variations with automatic metering
func (i *ImagesInterface) NewVariation(ctx context.Context, params openai.ImageNewVariationParams, opts ...option.RequestOption) (*openai.ImagesResponse, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Get requested image count from params (default is 1)
	requestedCount := 1
	if params.N.Value != 0 {
		requestedCount = int(params.N.Value)
	}

	// Call OpenAI Images Variation API
	resp, err := i.client.Images.NewVariation(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		i.parent.wg.Add(1)
		go func() {
			defer i.parent.wg.Done()
			i.sendImageMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error(), requestedCount)
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	i.parent.wg.Add(1)
	go func() {
		defer i.parent.wg.Done()
		i.sendImageMeteringData(ctx, resp, string(params.Model), metadata, duration, requestTime, requestedCount)
	}()

	return resp, nil
}

// sendImageMeteringData sends metering data for image generation
func (i *ImagesInterface) sendImageMeteringData(ctx context.Context, resp *openai.ImagesResponse, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, requestedCount int) {
	// Build payload
	payload := i.buildImageMeteringPayload(resp, model, metadata, duration, requestTime, requestedCount)

	Debug("[METERING] Sending image metering data...")
	if err := i.sendImageMeteringRequest(payload); err != nil {
		Error("Failed to send image metering data: %v", err)
	} else {
		Debug("[METERING] Image metering data sent successfully")
	}
}

// sendImageMeteringForError sends metering data for failed image generation
func (i *ImagesInterface) sendImageMeteringForError(ctx context.Context, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string, requestedCount int) {
	payload := i.buildImageErrorMeteringPayload(model, metadata, duration, requestTime, errorReason, requestedCount)

	Debug("[METERING] Sending image error metering data...")
	if err := i.sendImageMeteringRequest(payload); err != nil {
		Error("Failed to send image error metering data: %v", err)
	} else {
		Debug("[METERING] Image error metering data sent successfully")
	}
}

// buildImageMeteringPayload builds the metering payload for image generation
func (i *ImagesInterface) buildImageMeteringPayload(resp *openai.ImagesResponse, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, requestedCount int) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	// Count actual images returned
	actualCount := 0
	if resp != nil && resp.Data != nil {
		actualCount = len(resp.Data)
	}

	// Build attributes for image dimensions (not billing fields)
	attributes := make(map[string]interface{})
	if resp != nil && len(resp.Data) > 0 {
		// Try to get dimensions from response size field
		if resp.Size != "" {
			attributes["size"] = string(resp.Size)
		}
		if resp.Quality != "" {
			attributes["quality"] = string(resp.Quality)
		}
		if resp.OutputFormat != "" {
			attributes["outputFormat"] = string(resp.OutputFormat)
		}
	}

	payload := map[string]interface{}{
		"stopReason":          "END",
		"costType":            "AI",
		"operationType":       "IMAGE",
		"model":               model,
		"provider":            "OPENAI",
		"transactionId":       generateRequestID(),
		"requestTime":         requestTimeISO,
		"responseTime":        responseTimeISO,
		"requestDuration":     duration.Milliseconds(),
		"middlewareSource":    GetMiddlewareSource(),
		// Image-specific billing fields (TOP LEVEL per API contract)
		"actualImageCount":    actualCount,
		"requestedImageCount": requestedCount,
	}

	// Add attributes if any
	if len(attributes) > 0 {
		payload["attributes"] = attributes
	}

	// Add token usage if available (gpt-image-1)
	if resp != nil && resp.Usage.InputTokens > 0 {
		payload["inputTokenCount"] = resp.Usage.InputTokens
		payload["outputTokenCount"] = resp.Usage.OutputTokens
		payload["totalTokenCount"] = resp.Usage.TotalTokens
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// buildImageErrorMeteringPayload builds the metering payload for failed image generation
func (i *ImagesInterface) buildImageErrorMeteringPayload(model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string, requestedCount int) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	payload := map[string]interface{}{
		"stopReason":          "ERROR",
		"costType":            "AI",
		"operationType":       "IMAGE",
		"model":               model,
		"provider":            "OPENAI",
		"transactionId":       generateRequestID(),
		"requestTime":         requestTimeISO,
		"responseTime":        responseTimeISO,
		"requestDuration":     duration.Milliseconds(),
		"middlewareSource":    GetMiddlewareSource(),
		"errorReason":         errorReason,
		// Image-specific billing fields
		"actualImageCount":    0,
		"requestedImageCount": requestedCount,
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// sendImageMeteringRequest sends the metering request to the images endpoint
func (i *ImagesInterface) sendImageMeteringRequest(payload map[string]interface{}) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := i.doImageMeteringRequest(payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if IsValidationError(err) {
			return err
		}
	}

	return NewMeteringError(fmt.Sprintf("image metering failed after %d retries", maxRetries), lastErr)
}

// doImageMeteringRequest sends a single metering request
func (i *ImagesInterface) doImageMeteringRequest(payload map[string]interface{}) error {
	baseURL := i.config.ReveniumBaseURL
	if baseURL == "" {
		baseURL = "https://api.revenium.ai"
	}
	url := baseURL + "/meter/v2/ai/images"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return NewMeteringError("failed to marshal image metering payload", err)
	}

	Debug("Sending image metering request to %s", url)
	Debug("Payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewMeteringError("failed to create image metering request", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-api-key", i.config.ReveniumAPIKey)
	req.Header.Set("User-Agent", "revenium-middleware-openai-go/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NewNetworkError("image metering request failed", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return NewValidationError(
				fmt.Sprintf("image metering API returned %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}
		return NewMeteringError("image metering API error", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	Debug("Image metering request successful")
	return nil
}
