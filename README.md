# Revenium Middleware for OpenAI (Go)

A lightweight, production-ready middleware that adds **Revenium metering and tracking** to OpenAI and Azure OpenAI API calls.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)](https://golang.org/)
[![Documentation](https://img.shields.io/badge/docs-revenium.io-blue)](https://docs.revenium.io)
[![Website](https://img.shields.io/badge/website-revenium.ai-blue)](https://www.revenium.ai)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Dual Provider Support** - Works with both OpenAI native API and Azure OpenAI
- **Automatic Metering** - Tracks all API calls with detailed usage metrics
- **Streaming Support** - Full support for streaming chat completions
- **Custom Metadata** - Add custom tracking metadata to any request
- **Production Ready** - Battle-tested and optimized for production use
- **Type Safe** - Built with Go's strong typing system

## Getting Started (5 minutes)

### Step 1: Create Your Project

```bash
mkdir my-openai-project
cd my-openai-project
go mod init my-openai-project
```

### Step 2: Install Dependencies

```bash
go get github.com/openai/openai-go/v3
go get github.com/revenium/revenium-middleware-openai-go
go mod tidy
```

This installs both the OpenAI SDK and the Revenium middleware.

### Step 3: Create Environment File

Create a `.env` file in your project root with your API keys:

```bash
# .env

# Revenium Configuration (Required)
REVENIUM_METERING_API_KEY=hak_your_api_key_here
REVENIUM_METERING_BASE_URL=https://api.revenium.ai

# OpenAI Configuration (Required for OpenAI native API)
OPENAI_API_KEY=sk-your_key_here

# Optional OpenAI Configuration
OPENAI_ORG_ID=org-your_organization_id

# Azure OpenAI Configuration (Required for Azure OpenAI support)
AZURE_OPENAI_API_KEY=your_azure_openai_api_key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_VERSION=your_azure_api_version

# Middleware Configuration (Optional)
REVENIUM_AZURE_DISABLE=1                    # Set to 1 to disable Azure OpenAI support
REVENIUM_DEBUG=false                        # Set to true to enable debug logging
```

**Replace the placeholder values with your actual keys!**

For a complete list of all available environment variables, see the [Configuration Options](#environment-variables) section below.

> **Note**: `REVENIUM_METERING_BASE_URL` defaults to `https://api.revenium.ai` and doesn't need to be set unless using a different environment.

## Examples

This repository includes runnable examples demonstrating how to use the Revenium middleware with OpenAI and Azure OpenAI:

- [OpenAI Examples](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/openai)
- [Azure OpenAI Examples](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/azure)

### Run examples after setup:

```bash
# OpenAI Examples
make run-openai-getting-started
make run-openai-basic
make run-openai-streaming
make run-openai-metadata

# Azure OpenAI Examples
make run-azure-getting-started
make run-azure-basic
make run-azure-streaming
make run-azure-metadata
```

## What Gets Tracked

The middleware automatically captures comprehensive usage data:

### **Usage Metrics (Automatic)**

- **Token Counts** - Input tokens, output tokens, total tokens, cached tokens
- **Model Information** - Model name, provider (OpenAI or Azure OpenAI)
- **Request Timing** - Request duration, response time, time to first token (streaming)
- **Streaming Metrics** - Chunk count, streaming duration
- **Stop Reason** - Automatically mapped from OpenAI's `finish_reason` to Revenium's standardized stop reasons
- **Temperature** - Automatically extracted from request parameters
- **Error Tracking** - Failed requests with error reasons

### **Business Context (Optional via Metadata)**

- **Organization Data** - Organization ID, product ID, subscription ID
- **Task Classification** - Task type, agent identifier
- **Tracing** - Trace ID for distributed tracing
- **Quality Metrics** - Response quality score
- **Subscriber Information** - Complete subscriber object with ID, email, and credentials

### **Technical Details**

- **API Endpoints** - Chat completions (streaming and non-streaming)
- **Request Types** - Streaming vs non-streaming
- **Provider Detection** - Automatic detection of OpenAI vs Azure OpenAI
- **Middleware Source** - Automatically set to "go"
- **Transaction ID** - Unique ID for each request

## Environment Variables

### Required

```bash
REVENIUM_METERING_API_KEY=your-api-key-here
OPENAI_API_KEY=your-api-key-here  # For OpenAI native API
REVENIUM_AZURE_DISABLE=1  # Set to 1 to disable Azure OpenAI support
```

### Optional

```bash
REVENIUM_DEBUG=false  # Set to true to enable debug logging
REVENIUM_METERING_BASE_URL=https://api.revenium.ai  # Optional, defaults to https://api.revenium.ai
OPENAI_ORG_ID=org-your_organization_id  # Optional OpenAI organization ID
```

### Required for Azure OpenAI

```bash
AZURE_OPENAI_API_KEY=your_azure_openai_api_key  # Required for Azure OpenAI
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/  # Required for Azure OpenAI
AZURE_OPENAI_API_VERSION=your_azure_api_version  # Required for Azure OpenAI

```

## Azure OpenAI Configuration

To use Azure OpenAI with the middleware:

### 1. Configure Azure OpenAI

```bash
# 1. Create an Azure OpenAI resource in Azure Portal
# 2. Deploy a model (e.g., gpt-4o) and note the deployment name
# 3. Set the following environment variables:
AZURE_OPENAI_API_KEY=your_azure_openai_api_key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_VERSION=your_azure_api_version
```

### 2. Enable Azure OpenAI

```bash
REVENIUM_AZURE_DISABLE=0  # Set to 0 to enable Azure OpenAI support
```

### 3. Important: Use Deployment Names

**When using Azure OpenAI, you must pass your Azure deployment name (not the OpenAI model name) in the `Model` parameter:**

```go
params := openai.ChatCompletionNewParams{
    Model: "your-azure-deployment-name", // e.g., "gpt-4o-deployment"
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Hello!"),
    },
}
```

**Note:** If you have Azure enabled (`REVENIUM_AZURE_DISABLE=0`) but use an OpenAI model name (like `"gpt-4o"`) instead of your Azure deployment name, the request will fail with a "DeploymentNotFound" error.

See the getting started example for Azure OpenAI [here](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/azure/getting-started)

## API Overview

- **`Initialize()`** - Initialize the middleware from environment variables
- **`GetClient()`** - Get the global Revenium client instance
- **`NewReveniumOpenAI(cfg)`** - Create a new client with explicit configuration
- **`WithUsageMetadata(ctx, metadata)`** - Add custom metadata to a request context
- **`Close()`** - Wait for all pending metering requests to complete

**For complete API documentation and usage examples, see [`examples/README.md`](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/README.md).**

## Metadata Fields

The middleware supports the following optional metadata fields for tracking:

| Field                   | Type   | Description                                                |
| ----------------------- | ------ | ---------------------------------------------------------- |
| `traceId`               | string | Unique identifier for session or conversation tracking     |
| `taskType`              | string | Type of AI task being performed (e.g., "chat", "analysis") |
| `agent`                 | string | AI agent or bot identifier                                 |
| `organizationId`        | string | Organization or company identifier                         |
| `productId`             | string | Your product or feature identifier                         |
| `subscriptionId`        | string | Subscription plan identifier                               |
| `responseQualityScore`  | number | Custom quality rating (0.0-1.0)                            |
| `subscriber.id`         | string | Unique user identifier                                     |
| `subscriber.email`      | string | User email address                                         |
| `subscriber.credential` | object | Authentication credential (`name` and `value` fields)      |

**All metadata fields are optional.** For complete metadata documentation and usage examples, see:

- [`examples/README.md`](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/README.md) - All usage examples
- [Revenium API Reference](https://revenium.readme.io/reference/meter_ai_completion) - Complete API documentation

## How It Works

1. **Initialize**: Call `Initialize()` to set up the middleware with your configuration
2. **Get Client**: Call `GetClient()` to get a wrapped OpenAI client instance
3. **Make Requests**: Use the client normally - all requests are automatically tracked
4. **Async Tracking**: Usage data is sent to Revenium in the background (fire-and-forget)
5. **Transparent Response**: Original OpenAI responses are returned unchanged
6. **Graceful Shutdown**: Call `Close()` to wait for all pending metering requests

The middleware never blocks your application - if Revenium tracking fails, your OpenAI requests continue normally.

**Supported APIs:**

- Chat Completions API (`client.Chat().Completions().New()`)
- Streaming API (`client.Chat().Completions().NewStreaming()`)
- Both OpenAI native API and Azure OpenAI providers

## Troubleshooting

### Common Issues

**No tracking data appears:**

1. Verify environment variables are set correctly in `.env`
2. Enable debug logging by setting `REVENIUM_DEBUG=true` in `.env`
3. Check console for `[Revenium]` log messages
4. Verify your `REVENIUM_METERING_API_KEY` is valid

**Client not initialized error:**

- Make sure you call `Initialize()` before `GetClient()`
- Check that your `.env` file is in the project root
- Verify `REVENIUM_METERING_API_KEY` is set

**OpenAI API errors:**

- Verify `OPENAI_API_KEY` is set correctly
- Ensure you're using a valid model name (e.g., `gpt-4o`)
- Set `REVENIUM_AZURE_DISABLE=1` to use OpenAI native API

**Azure OpenAI API errors:**

- Verify all three Azure variables are set: `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_VERSION`
- Check that you're using your Azure deployment name (not OpenAI model name) in the `Model` parameter
- Set `REVENIUM_AZURE_DISABLE=0` to enable Azure OpenAI

**Wrong provider detected:**

- If you have both OpenAI and Azure OpenAI credentials configured, the middleware will auto-detect based on Azure credentials
- To force OpenAI native API, set `REVENIUM_AZURE_DISABLE=1`
- To force Azure OpenAI, set `REVENIUM_AZURE_DISABLE=0` (or leave unset)

**"DeploymentNotFound" error with Azure:**

- You're using an OpenAI model name (like `"gpt-4o"`) instead of your Azure deployment name
- Solution: Use your Azure deployment name in the `Model` parameter, OR disable Azure with `REVENIUM_AZURE_DISABLE=1`

### Debug Mode

Enable detailed logging by adding to your `.env`:

```env
REVENIUM_DEBUG=true
```

### Getting Help

If issues persist:

1. Enable debug logging (`REVENIUM_DEBUG=true`)
2. Check the [`examples/`](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples) directory for working examples
3. Review [`examples/README.md`](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples/README.md) for detailed setup instructions
4. Contact support@revenium.io with debug logs

## Supported Models

This middleware works with any OpenAI or Azure OpenAI model. For the complete model list, see:

- [OpenAI Models Documentation](https://platform.openai.com/docs/models)
- [Azure OpenAI Models Documentation](https://learn.microsoft.com/en-us/azure/ai-services/openai/concepts/models)

### API Support Matrix

The following table shows what has been tested and verified with working examples:

| Feature               | OpenAI | Azure OpenAI |
| --------------------- | ------ | ------------ |
| **Basic Usage**       | Yes    | Yes          |
| **Streaming**         | Yes    | Yes          |
| **Metadata Tracking** | Yes    | Yes          |
| **Token Counting**    | Yes    | Yes          |

**Note:** "Yes" = Tested with working examples in [`examples/`](https://github.com/revenium/revenium-middleware-openai-go/tree/HEAD/examples) directory

## Requirements

- Go 1.22+
- Revenium API key
- OpenAI API key (for OpenAI native) OR Azure OpenAI credentials (for Azure OpenAI)

## Documentation

For detailed documentation, visit [docs.revenium.io](https://docs.revenium.io)

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/revenium/revenium-middleware-openai-go/blob/HEAD/LICENSE) file for details.

## Support

For issues, feature requests, or contributions:

- **GitHub Repository**: [revenium/revenium-middleware-openai-go](https://github.com/revenium/revenium-middleware-openai-go)
- **Issues**: [Report bugs or request features](https://github.com/revenium/revenium-middleware-openai-go/issues)
- **Documentation**: [docs.revenium.io](https://docs.revenium.io)
- **Contact**: Reach out to the Revenium team for additional support

---

**Built by Revenium**
