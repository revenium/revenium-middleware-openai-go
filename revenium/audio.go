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

// Audio returns the audio interface for TTS and transcription with metering
func (r *ReveniumOpenAI) Audio() *AudioInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &AudioInterface{
		client:   r.client,
		config:   r.config,
		provider: r.provider,
		parent:   r,
	}
}

// AudioInterface provides methods for audio operations with metering
type AudioInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI
}

// Speech returns the speech interface for text-to-speech
func (a *AudioInterface) Speech() *SpeechInterface {
	return &SpeechInterface{
		client:   a.client,
		config:   a.config,
		provider: a.provider,
		parent:   a.parent,
	}
}

// Transcriptions returns the transcriptions interface for speech-to-text
func (a *AudioInterface) Transcriptions() *TranscriptionsInterface {
	return &TranscriptionsInterface{
		client:   a.client,
		config:   a.config,
		provider: a.provider,
		parent:   a.parent,
	}
}

// Translations returns the translations interface for speech translation
func (a *AudioInterface) Translations() *TranslationsInterface {
	return &TranslationsInterface{
		client:   a.client,
		config:   a.config,
		provider: a.provider,
		parent:   a.parent,
	}
}

// SpeechInterface provides methods for text-to-speech with metering
type SpeechInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI
}

// New creates speech from text with automatic metering
func (s *SpeechInterface) New(ctx context.Context, params openai.AudioSpeechNewParams, opts ...option.RequestOption) (*http.Response, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Calculate input character count for metering
	inputCharCount := len(params.Input)

	// Call OpenAI TTS API
	resp, err := s.client.Audio.Speech.New(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		s.parent.wg.Add(1)
		go func() {
			defer s.parent.wg.Done()
			s.sendSpeechMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error(), inputCharCount)
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	// Note: We can't easily determine audio duration from response without decoding
	// So we use input character count as a proxy for billing
	s.parent.wg.Add(1)
	go func() {
		defer s.parent.wg.Done()
		s.sendSpeechMeteringData(ctx, string(params.Model), metadata, duration, requestTime, inputCharCount)
	}()

	return resp, nil
}

// sendSpeechMeteringData sends metering data for TTS
func (s *SpeechInterface) sendSpeechMeteringData(ctx context.Context, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, inputCharCount int) {
	payload := s.buildSpeechMeteringPayload(model, metadata, duration, requestTime, inputCharCount)

	Debug("[METERING] Sending speech metering data...")
	if err := s.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send speech metering data: %v", err)
	} else {
		Debug("[METERING] Speech metering data sent successfully")
	}
}

// sendSpeechMeteringForError sends metering data for failed TTS
func (s *SpeechInterface) sendSpeechMeteringForError(ctx context.Context, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string, inputCharCount int) {
	payload := s.buildSpeechErrorMeteringPayload(model, metadata, duration, requestTime, errorReason, inputCharCount)

	Debug("[METERING] Sending speech error metering data...")
	if err := s.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send speech error metering data: %v", err)
	} else {
		Debug("[METERING] Speech error metering data sent successfully")
	}
}

// buildSpeechMeteringPayload builds the metering payload for TTS
func (s *SpeechInterface) buildSpeechMeteringPayload(model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, inputCharCount int) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	payload := map[string]interface{}{
		"stopReason":       "END",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "tts", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
		// TTS billing field (top-level per API contract)
		"characterCount": inputCharCount,
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// buildSpeechErrorMeteringPayload builds the metering payload for failed TTS
func (s *SpeechInterface) buildSpeechErrorMeteringPayload(model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string, inputCharCount int) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	payload := map[string]interface{}{
		"stopReason":       "ERROR",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "tts", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
		"errorReason":      errorReason,
		// TTS billing field (top-level per API contract)
		"characterCount": inputCharCount,
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// sendAudioMeteringRequest sends the metering request to the audio endpoint
func (s *SpeechInterface) sendAudioMeteringRequest(payload map[string]interface{}) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := s.doAudioMeteringRequest(payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if IsValidationError(err) {
			return err
		}
	}

	return NewMeteringError(fmt.Sprintf("audio metering failed after %d retries", maxRetries), lastErr)
}

// doAudioMeteringRequest sends a single metering request
func (s *SpeechInterface) doAudioMeteringRequest(payload map[string]interface{}) error {
	baseURL := s.config.ReveniumBaseURL
	if baseURL == "" {
		baseURL = "https://api.revenium.ai"
	}
	url := baseURL + "/meter/v2/ai/audio"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return NewMeteringError("failed to marshal audio metering payload", err)
	}

	Debug("Sending audio metering request to %s", url)
	Debug("Payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewMeteringError("failed to create audio metering request", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-api-key", s.config.ReveniumAPIKey)
	req.Header.Set("User-Agent", "revenium-middleware-openai-go/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NewNetworkError("audio metering request failed", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return NewValidationError(
				fmt.Sprintf("audio metering API returned %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}
		return NewMeteringError("audio metering API error", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	Debug("Audio metering request successful")
	return nil
}

// TranscriptionsInterface provides methods for speech-to-text with metering
type TranscriptionsInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI
}

// New creates a transcription from audio with automatic metering
func (t *TranscriptionsInterface) New(ctx context.Context, params openai.AudioTranscriptionNewParams, opts ...option.RequestOption) (*openai.AudioTranscriptionNewResponseUnion, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Call OpenAI Whisper API
	resp, err := t.client.Audio.Transcriptions.New(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		t.parent.wg.Add(1)
		go func() {
			defer t.parent.wg.Done()
			t.sendTranscriptionMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error())
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	t.parent.wg.Add(1)
	go func() {
		defer t.parent.wg.Done()
		t.sendTranscriptionMeteringData(ctx, resp, string(params.Model), metadata, duration, requestTime)
	}()

	return resp, nil
}

// sendTranscriptionMeteringData sends metering data for transcription
func (t *TranscriptionsInterface) sendTranscriptionMeteringData(ctx context.Context, resp *openai.AudioTranscriptionNewResponseUnion, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time) {
	payload := t.buildTranscriptionMeteringPayload(resp, model, metadata, duration, requestTime)

	Debug("[METERING] Sending transcription metering data...")
	if err := t.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send transcription metering data: %v", err)
	} else {
		Debug("[METERING] Transcription metering data sent successfully")
	}
}

// sendTranscriptionMeteringForError sends metering data for failed transcription
func (t *TranscriptionsInterface) sendTranscriptionMeteringForError(ctx context.Context, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string) {
	payload := t.buildTranscriptionErrorMeteringPayload(model, metadata, duration, requestTime, errorReason)

	Debug("[METERING] Sending transcription error metering data...")
	if err := t.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send transcription error metering data: %v", err)
	} else {
		Debug("[METERING] Transcription error metering data sent successfully")
	}
}

// buildTranscriptionMeteringPayload builds the metering payload for transcription
func (t *TranscriptionsInterface) buildTranscriptionMeteringPayload(resp *openai.AudioTranscriptionNewResponseUnion, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	// Transcription uses audio endpoint with durationSeconds for per-minute billing
	payload := map[string]interface{}{
		"stopReason":       "END",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "transcription", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
	}

	// Extract durationSeconds from metadata if provided, otherwise use 0
	// Users can provide this via WithUsageMetadata(ctx, map[string]interface{}{"durationSeconds": 60.0})
	if durationSecs, ok := metadata["durationSeconds"].(float64); ok {
		payload["durationSeconds"] = durationSecs
	} else if durationSecs, ok := metadata["audioDuration"].(float64); ok {
		payload["durationSeconds"] = durationSecs
	} else {
		// Default to 0 - backend may accept or require this field
		payload["durationSeconds"] = float64(0)
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// buildTranscriptionErrorMeteringPayload builds the metering payload for failed transcription
func (t *TranscriptionsInterface) buildTranscriptionErrorMeteringPayload(model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	payload := map[string]interface{}{
		"stopReason":       "ERROR",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "transcription", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
		"errorReason":      errorReason,
		"durationSeconds":  float64(0), // Required for audio billing
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// sendAudioMeteringRequest sends the metering request to the audio endpoint
func (t *TranscriptionsInterface) sendAudioMeteringRequest(payload map[string]interface{}) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := t.doAudioMeteringRequest(payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if IsValidationError(err) {
			return err
		}
	}

	return NewMeteringError(fmt.Sprintf("transcription metering failed after %d retries", maxRetries), lastErr)
}

// doAudioMeteringRequest sends a single metering request to audio endpoint
func (t *TranscriptionsInterface) doAudioMeteringRequest(payload map[string]interface{}) error {
	baseURL := t.config.ReveniumBaseURL
	if baseURL == "" {
		baseURL = "https://api.revenium.ai"
	}
	url := baseURL + "/meter/v2/ai/audio"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return NewMeteringError("failed to marshal transcription metering payload", err)
	}

	Debug("Sending transcription metering request to %s", url)
	Debug("Payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewMeteringError("failed to create transcription metering request", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-api-key", t.config.ReveniumAPIKey)
	req.Header.Set("User-Agent", "revenium-middleware-openai-go/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NewNetworkError("transcription metering request failed", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return NewValidationError(
				fmt.Sprintf("transcription metering API returned %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}
		return NewMeteringError("transcription metering API error", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	Debug("Transcription metering request successful")
	return nil
}

// TranslationsInterface provides methods for audio translation with metering
type TranslationsInterface struct {
	client   openai.Client
	config   *Config
	provider Provider
	parent   *ReveniumOpenAI
}

// New creates a translation from audio with automatic metering
func (t *TranslationsInterface) New(ctx context.Context, params openai.AudioTranslationNewParams, opts ...option.RequestOption) (*openai.Translation, error) {
	// Extract metadata from context
	metadata := GetUsageMetadata(ctx)

	// Record start time
	requestTime := time.Now()

	// Call OpenAI Whisper Translation API
	resp, err := t.client.Audio.Translations.New(ctx, params, opts...)
	if err != nil {
		duration := time.Since(requestTime)
		t.parent.wg.Add(1)
		go func() {
			defer t.parent.wg.Done()
			t.sendTranslationMeteringForError(ctx, string(params.Model), metadata, duration, requestTime, err.Error())
		}()
		return nil, err
	}

	// Calculate duration
	duration := time.Since(requestTime)

	// Send metering data asynchronously
	t.parent.wg.Add(1)
	go func() {
		defer t.parent.wg.Done()
		t.sendTranslationMeteringData(ctx, resp, string(params.Model), metadata, duration, requestTime)
	}()

	return resp, nil
}

// sendTranslationMeteringData sends metering data for translation
func (t *TranslationsInterface) sendTranslationMeteringData(ctx context.Context, resp *openai.Translation, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time) {
	payload := t.buildTranslationMeteringPayload(resp, model, metadata, duration, requestTime)

	Debug("[METERING] Sending translation metering data...")
	if err := t.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send translation metering data: %v", err)
	} else {
		Debug("[METERING] Translation metering data sent successfully")
	}
}

// sendTranslationMeteringForError sends metering data for failed translation
func (t *TranslationsInterface) sendTranslationMeteringForError(ctx context.Context, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string) {
	payload := t.buildTranslationErrorMeteringPayload(model, metadata, duration, requestTime, errorReason)

	Debug("[METERING] Sending translation error metering data...")
	if err := t.sendAudioMeteringRequest(payload); err != nil {
		Error("Failed to send translation error metering data: %v", err)
	} else {
		Debug("[METERING] Translation error metering data sent successfully")
	}
}

// buildTranslationMeteringPayload builds the metering payload for translation
func (t *TranslationsInterface) buildTranslationMeteringPayload(resp *openai.Translation, model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	// Translation uses audio endpoint with durationSeconds for per-minute billing
	payload := map[string]interface{}{
		"stopReason":       "END",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "translation", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
	}

	// Extract durationSeconds from metadata if provided, otherwise use 0
	// Users can provide this via WithUsageMetadata(ctx, map[string]interface{}{"durationSeconds": 60.0})
	if durationSecs, ok := metadata["durationSeconds"].(float64); ok {
		payload["durationSeconds"] = durationSecs
	} else if durationSecs, ok := metadata["audioDuration"].(float64); ok {
		payload["durationSeconds"] = durationSecs
	} else {
		// Default to 0 - backend may accept or require this field
		payload["durationSeconds"] = float64(0)
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// buildTranslationErrorMeteringPayload builds the metering payload for failed translation
func (t *TranslationsInterface) buildTranslationErrorMeteringPayload(model string, metadata map[string]interface{}, duration time.Duration, requestTime time.Time, errorReason string) map[string]interface{} {
	responseTime := time.Now().UTC()
	responseTimeISO := responseTime.Format(time.RFC3339)
	requestTimeISO := requestTime.UTC().Format(time.RFC3339)

	payload := map[string]interface{}{
		"stopReason":       "ERROR",
		"costType":         "AI",
		"operationType":    "AUDIO",
		"operationSubtype": "translation", // lowercase per API contract
		"model":            model,
		"provider":         "OPENAI",
		"transactionId":    generateRequestID(),
		"requestTime":      requestTimeISO,
		"responseTime":     responseTimeISO,
		"requestDuration":  duration.Milliseconds(),
		"middlewareSource": GetMiddlewareSource(),
		"errorReason":      errorReason,
		"durationSeconds":  float64(0), // Required for audio billing
	}

	addMetadataToPayload(payload, metadata)
	return payload
}

// sendAudioMeteringRequest sends the metering request to the audio endpoint
func (t *TranslationsInterface) sendAudioMeteringRequest(payload map[string]interface{}) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := t.doAudioMeteringRequest(payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if IsValidationError(err) {
			return err
		}
	}

	return NewMeteringError(fmt.Sprintf("translation metering failed after %d retries", maxRetries), lastErr)
}

// doAudioMeteringRequest sends a single metering request to audio endpoint
func (t *TranslationsInterface) doAudioMeteringRequest(payload map[string]interface{}) error {
	baseURL := t.config.ReveniumBaseURL
	if baseURL == "" {
		baseURL = "https://api.revenium.ai"
	}
	url := baseURL + "/meter/v2/ai/audio"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return NewMeteringError("failed to marshal translation metering payload", err)
	}

	Debug("Sending translation metering request to %s", url)
	Debug("Payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewMeteringError("failed to create translation metering request", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-api-key", t.config.ReveniumAPIKey)
	req.Header.Set("User-Agent", "revenium-middleware-openai-go/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NewNetworkError("translation metering request failed", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return NewValidationError(
				fmt.Sprintf("translation metering API returned %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}
		return NewMeteringError("translation metering API error", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	Debug("Translation metering request successful")
	return nil
}
