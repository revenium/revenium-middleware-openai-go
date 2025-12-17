# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2025-12-16

### Added

- Initial release of Revenium OpenAI Go Middleware
- Support for OpenAI native API provider
- Support for Azure OpenAI provider
- Automatic provider detection based on configuration
- Initialize()/GetClient() pattern for client management
- Automatic .env file loading via godotenv
- Context-based metadata management with WithUsageMetadata()
- Comprehensive usage tracking and metering:
  - Token counts (prompt, completion, total, cached tokens)
  - Request timing (duration, time to first token for streaming)
  - Model information and provider detection
  - Stop reason mapping from OpenAI's finish_reason
  - Temperature and other parameters extraction
  - Error tracking with detailed error reasons
- Streaming support with automatic metrics tracking:
  - Chunk count tracking
  - Streaming duration measurement
  - Time to first token calculation
  - Accumulated response tracking
- Fire-and-forget metering (non-blocking background processing)
- Debug logging support via REVENIUM_DEBUG environment variable
- Complete metadata support:
  - Organization tracking (organizationId, productId, subscriptionId)
  - Task classification (taskType, agent)
  - Distributed tracing (traceId)
  - Quality metrics (responseQualityScore on 0.0-1.0 scale)
  - Subscriber information (complete object with id, email, credentials)
- Comprehensive examples:
  - OpenAI examples: getting-started, basic, streaming, metadata
  - Azure OpenAI examples: getting-started, basic, streaming, metadata
- Full documentation:
  - README.md with quick start and comprehensive guides
  - DEVELOPMENT.md for contributors and developers
  - CONTRIBUTING.md with contribution guidelines
  - CODE_OF_CONDUCT.md for community standards
  - SECURITY.md for security policy
  - Examples README with detailed example descriptions
- Makefile with convenient commands:
  - make install, test, lint, fmt, clean
  - make run-openai-\* for OpenAI examples
  - make run-azure-\* for Azure OpenAI examples
  - make build-examples for building all examples
- Stop reason mapping:
  - stop → END
  - length → LENGTH
  - content_filter → CONTENT_FILTER
  - tool_calls → TOOL_CALLS
  - function_call → FUNCTION_CALL
  - null/other → OTHER
- Proper error handling and recovery
- PII protection in debug logs (masked email addresses)
- Automatic transaction ID generation for request tracking
- Azure OpenAI configuration:
  - Required environment variables: AZURE_OPENAI_API_KEY, AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_VERSION
  - Optional disable flag: REVENIUM_AZURE_DISABLE
  - Automatic fallback to OpenAI native when Azure is disabled

### Technical Details

- Built with Go 1.22+
- Uses github.com/openai/openai-go/v3 SDK
- Clean architecture with separated concerns:
  - config.go - Configuration management
  - context.go - Context utilities
  - middleware.go - Core middleware logic
  - provider.go - Provider detection
  - stop_reason_mapper.go - Stop reason mapping
  - logger.go - Debug logging
  - errors.go - Error definitions
- Middleware source automatically set to "go"
- Cost type set to "AI"
- Operation type set to "CHAT"
- Provider values: "OPENAI" or "AZURE"
- Model source set to "OPENAI" for both providers
- ISO 8601 timestamp formatting for all time fields
- Comprehensive test coverage for stop reason mapping

### API Compliance

- Follows Revenium Metering API v2 specification
- Proper field naming and data types
- Correct stop reason enumeration
- Optional fields properly handled (undefined when not available)
- Token counts accurately reported from OpenAI's Usage object
- Streaming metrics properly calculated and reported
- responseQualityScore correctly scaled to 0.0-1.0 range

### Fixed

- Removed hardcoded Azure API version (now required via environment variable)
- Fixed responseQualityScore scale from 0-100 to 0.0-1.0
- Removed unnecessary provider verification logic from examples
- Cleaned up example structure with proper directory organization

[0.0.1]: https://github.com/revenium/revenium-middleware-openai-go/releases/tag/v0.0.1
