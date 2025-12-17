package revenium

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared/constant"
)

const middlewareSource = "go"

// ReveniumOpenAI is the main middleware client that wraps the OpenAI SDK
// and adds metering capabilities
type ReveniumOpenAI struct {
	client   openai.Client
	config   *Config
	provider Provider
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

var (
	globalClient *ReveniumOpenAI
	globalMu     sync.RWMutex
	initialized  bool
)

// Initialize sets up the global Revenium middleware with configuration
func Initialize(opts ...Option) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if initialized {
		return nil
	}

	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.loadFromEnv(); err != nil {
		Warn("Failed to load configuration from environment: %v", err)
	}

	Info("Initializing Revenium middleware...")

	if err := cfg.Validate(); err != nil {
		return err
	}

	provider := DetectProvider(cfg)
	clientOpts := buildClientOptions(cfg, provider)

	openaiClient := openai.NewClient(clientOpts...)

	globalClient = &ReveniumOpenAI{
		client:   openaiClient,
		config:   cfg,
		provider: provider,
	}

	initialized = true
	Info("Revenium middleware initialized successfully with provider: %s", provider)
	return nil
}

// IsInitialized checks if the middleware is properly initialized
func IsInitialized() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return initialized
}

// GetClient returns the global Revenium client
func GetClient() (*ReveniumOpenAI, error) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if !initialized {
		return nil, NewConfigError("middleware not initialized, call Initialize() first", nil)
	}

	return globalClient, nil
}

// NewReveniumOpenAI creates a new Revenium client with explicit configuration
func NewReveniumOpenAI(cfg *Config) (*ReveniumOpenAI, error) {
	if cfg == nil {
		return nil, NewConfigError("config cannot be nil", nil)
	}

	// Validate required fields
	if cfg.ReveniumAPIKey == "" {
		return nil, NewConfigError("REVENIUM_METERING_API_KEY is required", nil)
	}

	provider := DetectProvider(cfg)
	clientOpts := buildClientOptions(cfg, provider)
	openaiClient := openai.NewClient(clientOpts...)

	return &ReveniumOpenAI{
		client:   openaiClient,
		config:   cfg,
		provider: provider,
	}, nil
}

// buildClientOptions builds OpenAI client options based on provider
func buildClientOptions(cfg *Config, provider Provider) []option.RequestOption {
	clientOpts := []option.RequestOption{}

	if provider == ProviderAzure {
		if cfg.AzureEndpoint != "" && cfg.AzureAPIVersion != "" {
			clientOpts = append(clientOpts, azure.WithEndpoint(cfg.AzureEndpoint, cfg.AzureAPIVersion))
			if cfg.AzureAPIKey != "" {
				clientOpts = append(clientOpts, azure.WithAPIKey(cfg.AzureAPIKey))
			}
			Info("Configured Azure OpenAI with endpoint: %s, API version: %s", cfg.AzureEndpoint, cfg.AzureAPIVersion)
		}
	} else {
		if cfg.OpenAIAPIKey != "" {
			clientOpts = append(clientOpts, option.WithAPIKey(cfg.OpenAIAPIKey))
		}
		if cfg.OpenAIOrgID != "" {
			clientOpts = append(clientOpts, option.WithOrganization(cfg.OpenAIOrgID))
		}
		if cfg.BaseURL != "" {
			clientOpts = append(clientOpts, option.WithBaseURL(cfg.BaseURL))
		}
	}

	return clientOpts
}

// GetConfig returns the configuration
func (r *ReveniumOpenAI) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// GetProvider returns the detected provider
func (r *ReveniumOpenAI) GetProvider() Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.provider
}

// GetOpenAIClient returns the underlying OpenAI client
func (r *ReveniumOpenAI) GetOpenAIClient() openai.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// Chat returns the chat interface for creating chat completions
func (r *ReveniumOpenAI) Chat() *ChatInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &ChatInterface{
		client:   r.client,
		config:   r.config,
		provider: r.provider,
		parent:   r,
	}
}

func (r *ReveniumOpenAI) Flush() {
	Debug("Flushing pending metering requests...")
	r.wg.Wait()
	Debug("All metering requests completed")
}

func (r *ReveniumOpenAI) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Flush()
	return nil
}

// ChatInterface provides methods for creating chat completions with metering
type ChatInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI // Reference to parent for WaitGroup access
}

// Completions returns the completions interface
func (c *ChatInterface) Completions() *CompletionsInterface {
	return &CompletionsInterface{
		client:   c.client,
		config:   c.config,
		provider: c.provider,
		parent:   c.parent,
	}
}

// CompletionsInterface provides methods for creating chat completions
type CompletionsInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI // Reference to parent for WaitGroup access
}

// New creates a chat completion with automatic metering
func (c *CompletionsInterface) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Call the appropriate provider
	switch c.provider {
	case ProviderOpenAI:
		return c.createCompletionOpenAI(ctx, params, metadata)
	case ProviderAzure:
		return c.createCompletionAzure(ctx, params, metadata)
	default:
		return nil, NewProviderError("unknown provider", fmt.Errorf("provider: %v", c.provider))
	}
}

// NewStreaming creates a streaming chat completion with automatic metering
// Returns a StreamingWrapper that intercepts the stream and sends metering data when closed
func (c *CompletionsInterface) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) (*StreamingWrapper, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Call the appropriate provider
	switch c.provider {
	case ProviderOpenAI:
		return c.createCompletionStreamingOpenAI(ctx, params, metadata)
	case ProviderAzure:
		return c.createCompletionStreamingAzure(ctx, params, metadata)
	default:
		return nil, NewProviderError("unknown provider", fmt.Errorf("provider: %v", c.provider))
	}
}

// createCompletionOpenAI creates a chat completion using OpenAI native API
func (c *CompletionsInterface) createCompletionOpenAI(ctx context.Context, params openai.ChatCompletionNewParams, metadata map[string]interface{}) (*openai.ChatCompletion, error) {
	// Record start time for duration calculation
	requestTime := time.Now()

	// Call OpenAI API
	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		// Send error metering data
		duration := time.Since(requestTime)
		c.parent.wg.Add(1)
		go func() {
			defer c.parent.wg.Done()
			c.sendMeteringDataForError(ctx, string(params.Model), metadata, false, duration, "OPENAI", requestTime, err.Error())
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// For non-streaming, completionStartTime is approximately the same as requestTime
	// timeToFirstToken is 0 for non-streaming
	c.parent.wg.Add(1)
	go func() {
		defer c.parent.wg.Done()
		c.sendMeteringData(ctx, resp, metadata, false, duration, "OPENAI", requestTime, nil, 0)
	}()

	return resp, nil
}

func (c *CompletionsInterface) createCompletionAzure(ctx context.Context, params openai.ChatCompletionNewParams, metadata map[string]interface{}) (*openai.ChatCompletion, error) {
	requestTime := time.Now()
	originalModel := string(params.Model)
	Debug("Using Azure deployment name '%s' from user", originalModel)

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		Warn("Azure request failed: %v, falling back to OpenAI", err)
		duration := time.Since(requestTime)
		c.parent.wg.Add(1)
		go func() {
			defer c.parent.wg.Done()
			c.sendMeteringDataForError(ctx, originalModel, metadata, false, duration, "AZURE", requestTime, err.Error())
		}()
		return c.createCompletionOpenAI(ctx, params, metadata)
	}

	duration := time.Since(requestTime)
	c.parent.wg.Add(1)
	go func() {
		defer c.parent.wg.Done()
		c.sendMeteringData(ctx, resp, metadata, false, duration, "AZURE", requestTime, nil, 0)
	}()

	return resp, nil
}

// createCompletionStreamingOpenAI creates a streaming chat completion using OpenAI native API
func (c *CompletionsInterface) createCompletionStreamingOpenAI(ctx context.Context, params openai.ChatCompletionNewParams, metadata map[string]interface{}) (*StreamingWrapper, error) {
	// Call OpenAI streaming API
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	// Prepare metadata with model information
	streamMetadata := make(map[string]interface{})
	if metadata != nil {
		// Copy all user-provided metadata
		for k, v := range metadata {
			streamMetadata[k] = v
		}
	}

	// Add model from request if not already in metadata
	if _, ok := streamMetadata["model"]; !ok {
		streamMetadata["model"] = string(params.Model)
	}

	// Wrap stream for metering tracking
	wrapper := &StreamingWrapper{
		stream:      stream,
		config:      c.config,
		metadata:    streamMetadata,
		startTime:   time.Now(),
		completions: c,
		model:       string(params.Model),
		provider:    "OPENAI",
		parent:      c.parent,
	}

	// Return the wrapper instead of the raw stream
	return wrapper, nil
}

func (c *CompletionsInterface) createCompletionStreamingAzure(ctx context.Context, params openai.ChatCompletionNewParams, metadata map[string]interface{}) (*StreamingWrapper, error) {
	originalModel := string(params.Model)
	Debug("Using Azure deployment name '%s' from user", originalModel)

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	streamMetadata := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			streamMetadata[k] = v
		}
	}

	if _, ok := streamMetadata["model"]; !ok {
		streamMetadata["model"] = originalModel
	}

	wrapper := &StreamingWrapper{
		stream:      stream,
		config:      c.config,
		metadata:    streamMetadata,
		startTime:   time.Now(),
		completions: c,
		model:       originalModel,
		provider:    "AZURE",
		parent:      c.parent,
	}

	return wrapper, nil
}

// StreamingWrapper wraps a streaming response to track tokens and send metering data
type StreamingWrapper struct {
	stream         *ssestream.Stream[openai.ChatCompletionChunk]
	config         *Config
	metadata       map[string]interface{}
	startTime      time.Time
	firstTokenTime *time.Time
	completions    *CompletionsInterface
	model          string
	provider       string
	parent         *ReveniumOpenAI // Reference to parent for WaitGroup access
	mu             sync.Mutex

	// Token tracking
	inputTokens         int64
	outputTokens        int64
	totalTokens         int64
	reasoningTokens     int64
	cacheReadTokens     int64
	cacheCreationTokens int64

	// Finish reason tracking
	finishReason string

	// System fingerprint tracking
	systemFingerprint string
}

func (c *CompletionsInterface) sendMeteringData(ctx context.Context, resp *openai.ChatCompletion, metadata map[string]interface{}, isStreamed bool, duration time.Duration, provider string, requestTime time.Time, completionStartTime *time.Time, timeToFirstToken int64) {
	payload := buildMeteringPayload(resp, metadata, isStreamed, duration, provider, requestTime, completionStartTime, timeToFirstToken)
	Debug("[METERING] About to send metering data...")
	if err := c.sendMeteringWithRetry(payload); err != nil {
		Error("Failed to send metering data: %v", err)
	} else {
		Debug("[METERING] Metering data sent successfully")
	}
}

func (c *CompletionsInterface) sendMeteringDataForError(ctx context.Context, model string, metadata map[string]interface{}, isStreamed bool, duration time.Duration, provider string, requestTime time.Time, errorReason string) {
	payload := buildErrorMeteringPayload(model, metadata, isStreamed, duration, provider, requestTime, errorReason)
	Debug("[METERING] About to send error metering data...")
	if err := c.sendMeteringWithRetry(payload); err != nil {
		Error("Failed to send error metering data: %v", err)
	} else {
		Debug("[METERING] Error metering data sent successfully")
	}
}

func buildErrorMeteringPayload(model string, metadata map[string]interface{}, isStreamed bool, duration time.Duration, provider string, requestTime time.Time, errorReason string) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	if provider == "" {
		provider = "OPENAI"
	}

	payload := map[string]interface{}{
		"stopReason":              "ERROR",
		"costType":                "AI",
		"isStreamed":              isStreamed,
		"operationType":           "CHAT",
		"inputTokenCount":         int64(0),
		"outputTokenCount":        int64(0),
		"reasoningTokenCount":     int64(0),
		"cacheCreationTokenCount": int64(0),
		"cacheReadTokenCount":     int64(0),
		"totalTokenCount":         int64(0),
		"model":                   model,
		"transactionId":           generateRequestID(),
		"responseTime":            responseTimeISO,
		"requestDuration":         duration.Milliseconds(),
		"provider":                provider,
		"requestTime":             requestTimeISO,
		"completionStartTime":     requestTimeISO,
		"timeToFirstToken":        int64(0),
		"middlewareSource":        middlewareSource,
		"errorReason":             errorReason,
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

func generateRequestID() string {
	now := time.Now().UnixNano()
	return fmt.Sprintf("%d-%d", now, now%1000000)
}

func addMetadataToPayload(payload map[string]interface{}, metadata map[string]interface{}) {
	if metadata == nil {
		return
	}
	metadataFields := []string{
		// Core tracking fields
		"organizationId", "productId", "taskType", "taskId", "agent", "subscriptionId",
		"traceId", "transactionId", "subscriber", "responseQualityScore",
		"modelSource", "temperature", "mediationLatency",
		// Trace visualization fields (distributed tracing)
		// NOTE: operationType is fixed (API only accepts: CHAT, GENERATE, EMBED, CLASSIFY, SUMMARIZE, TRANSLATE, OTHER)
		// NOTE: operationSubtype is auto-detected, not user-provided
		"traceType", "traceName", "environment", "region",
		"retryNumber", "credentialAlias", "parentTransactionId",
	}
	for _, field := range metadataFields {
		if value, ok := metadata[field]; ok {
			payload[field] = value
		}
	}
}

func buildMeteringPayload(resp *openai.ChatCompletion, metadata map[string]interface{}, isStreamed bool, duration time.Duration, provider string, requestTime time.Time, completionStartTime *time.Time, timeToFirstToken int64) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	completionStartTimeISO := requestTimeISO
	if completionStartTime != nil {
		completionStartTimeISO = completionStartTime.UTC().Format(time.RFC3339)
	}

	if provider == "" {
		provider = "OPENAI"
	}

	inputTokens := resp.Usage.PromptTokens
	outputTokens := resp.Usage.CompletionTokens
	totalTokens := resp.Usage.TotalTokens
	reasoningTokens := int64(0)
	cacheCreationTokens := int64(0)
	cacheReadTokens := int64(0)

	if resp.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
		reasoningTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
	}

	if resp.Usage.PromptTokensDetails.CachedTokens > 0 {
		cacheReadTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	openaiFinishReason := ""
	if len(resp.Choices) > 0 {
		openaiFinishReason = resp.Choices[0].FinishReason
	}
	stopReason := string(MapOpenAIFinishReason(openaiFinishReason, StopReasonEnd))

	payload := map[string]interface{}{
		"stopReason":              stopReason,
		"costType":                "AI",
		"isStreamed":              isStreamed,
		"operationType":           "CHAT",
		"inputTokenCount":         inputTokens,
		"outputTokenCount":        outputTokens,
		"reasoningTokenCount":     reasoningTokens,
		"cacheCreationTokenCount": cacheCreationTokens,
		"cacheReadTokenCount":     cacheReadTokens,
		"totalTokenCount":         totalTokens,
		"model":                   string(resp.Model),
		"transactionId":           generateRequestID(),
		"responseTime":            responseTimeISO,
		"requestDuration":         duration.Milliseconds(),
		"provider":                provider,
		"requestTime":             requestTimeISO,
		"completionStartTime":     completionStartTimeISO,
		"timeToFirstToken":        timeToFirstToken,
		"middlewareSource":        middlewareSource,
	}

	if resp.SystemFingerprint != "" {
		payload["systemFingerprint"] = resp.SystemFingerprint
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

func (c *CompletionsInterface) sendMeteringWithRetry(payload map[string]interface{}) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := c.sendMeteringRequest(payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if IsValidationError(err) {
			return err
		}
	}

	return NewMeteringError(fmt.Sprintf("metering failed after %d retries", maxRetries), lastErr)
}

func (c *CompletionsInterface) sendMeteringRequest(payload map[string]interface{}) error {
	baseURL := c.config.ReveniumBaseURL
	if baseURL == "" {
		baseURL = "https://api.revenium.ai"
	}
	url := baseURL + "/meter/v2/ai/completions"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return NewMeteringError("failed to marshal metering payload", err)
	}

	Debug("Sending metering request to %s", url)
	Debug("Payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewMeteringError("failed to create metering request", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-api-key", c.config.ReveniumAPIKey)
	req.Header.Set("User-Agent", "revenium-middleware-openai-go/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NewNetworkError("metering request failed", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return NewValidationError(
				fmt.Sprintf("metering API returned %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}
		return NewMeteringError("metering API error", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	Debug("Metering request successful")
	return nil
}

func (sw *StreamingWrapper) Next() bool {
	return sw.stream.Next()
}

func (sw *StreamingWrapper) Current() openai.ChatCompletionChunk {
	chunk := sw.stream.Current()

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.firstTokenTime == nil && len(chunk.Choices) > 0 {
		now := time.Now()
		sw.firstTokenTime = &now
	}

	if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
		sw.inputTokens = chunk.Usage.PromptTokens
		sw.outputTokens = chunk.Usage.CompletionTokens
		sw.totalTokens = chunk.Usage.TotalTokens

		if chunk.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			sw.reasoningTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
		}

		if chunk.Usage.PromptTokensDetails.CachedTokens > 0 {
			sw.cacheReadTokens = chunk.Usage.PromptTokensDetails.CachedTokens
		}
	}

	if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != "" {
		sw.finishReason = chunk.Choices[0].FinishReason
	}

	if chunk.SystemFingerprint != "" {
		sw.systemFingerprint = chunk.SystemFingerprint
	}

	return chunk
}

func (sw *StreamingWrapper) Err() error {
	return sw.stream.Err()
}

func (sw *StreamingWrapper) Close() error {
	err := sw.stream.Close()
	streamErr := sw.stream.Err()
	duration := time.Since(sw.startTime)

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if streamErr != nil {
		sw.parent.wg.Add(1)
		go func() {
			defer sw.parent.wg.Done()
			sw.completions.sendMeteringDataForError(
				context.Background(),
				sw.model,
				sw.metadata,
				true,
				duration,
				sw.provider,
				sw.startTime,
				streamErr.Error(),
			)
		}()
		return err
	}

	timeToFirstToken := int64(0)
	var completionStartTime *time.Time
	if sw.firstTokenTime != nil {
		timeToFirstToken = sw.firstTokenTime.Sub(sw.startTime).Milliseconds()
		completionStartTime = sw.firstTokenTime
	}

	finishReason := sw.finishReason
	if finishReason == "" {
		finishReason = "stop"
	}

	resp := &openai.ChatCompletion{
		ID:                generateRequestID(),
		Model:             sw.model,
		Created:           sw.startTime.Unix(),
		SystemFingerprint: sw.systemFingerprint,
		Usage: openai.CompletionUsage{
			PromptTokens:     sw.inputTokens,
			CompletionTokens: sw.outputTokens,
			TotalTokens:      sw.totalTokens,
			CompletionTokensDetails: openai.CompletionUsageCompletionTokensDetails{
				ReasoningTokens: sw.reasoningTokens,
			},
			PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
				CachedTokens: sw.cacheReadTokens,
			},
		},
		Choices: []openai.ChatCompletionChoice{
			{
				FinishReason: finishReason,
				Message: openai.ChatCompletionMessage{
					Role:    constant.Assistant("assistant"),
					Content: "",
				},
			},
		},
	}

	sw.parent.wg.Add(1)
	go func() {
		defer sw.parent.wg.Done()
		sw.completions.sendMeteringData(context.Background(), resp, sw.metadata, true, duration, sw.provider, sw.startTime, completionStartTime, timeToFirstToken)
	}()

	return err
}
