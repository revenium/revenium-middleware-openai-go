# Revenium OpenAI Go Middleware - Examples

This directory contains examples demonstrating how to use the Revenium OpenAI Go middleware with both OpenAI native API and Azure OpenAI.

## Prerequisites

Before running the examples, make sure you have:

1. **Go 1.22+** installed
2. **Revenium API Key** - Get one from [Revenium Dashboard](https://app.revenium.ai)
3. **OpenAI API Key** (for OpenAI examples) - Get one from [OpenAI Platform](https://platform.openai.com)
4. **Azure OpenAI credentials** (for Azure examples) - Set up at [Azure Portal](https://portal.azure.com)

## Setup

1. **Clone the repository** (if you haven't already):

   ```bash
   git clone https://github.com/revenium/revenium-middleware-openai-go.git
   cd revenium-middleware-openai-go
   ```

2. **Install dependencies**:

   ```bash
   go mod download
   ```

3. **Configure environment variables**:

   Create a `.env` file in the project root with your API keys:

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
   # All three variables below are REQUIRED when using Azure OpenAI
   AZURE_OPENAI_API_KEY=your_azure_openai_api_key
   AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
   AZURE_OPENAI_API_VERSION=your_azure_api_version

   # Middleware Configuration (Optional)
   REVENIUM_AZURE_DISABLE=1                    # Set to 1 to disable Azure OpenAI support
   REVENIUM_DEBUG=false                        # Set to true to enable debug logging
   ```

   **Note:** The middleware automatically loads `.env` files via `Initialize()`, so no additional configuration is needed.

## Examples

### OpenAI Native API Examples

#### 1. Getting Started

**File:** `openai/getting-started/main.go`

The simplest example to get you started with Revenium tracking:

- Initialize the middleware
- Create a basic chat completion request
- Display response and usage metrics

**Run:**

```bash
make run-openai-getting-started
# or
go run examples/openai/getting-started/main.go
```

**What it does:**

- Loads configuration from environment variables
- Creates a simple chat completion request
- Automatically sends metering data to Revenium API
- Displays the response and token usage

---

#### 2. Basic Usage

**File:** `openai/basic/main.go`

Demonstrates standard OpenAI usage:

- Chat completion with metadata
- Simple metadata tracking

**Run:**

```bash
make run-openai-basic
# or
go run examples/openai/basic/main.go
```

**What it does:**

- Creates chat completion with metadata tracking
- Demonstrates basic metadata usage
- Shows token counting and usage metrics

---

#### 3. Streaming

**File:** `openai/streaming/main.go`

Demonstrates streaming responses:

- Real-time token streaming
- Accumulating responses
- Streaming metrics

**Run:**

```bash
make run-openai-streaming
# or
go run examples/openai/streaming/main.go
```

**What it does:**

- Creates a streaming chat completion request
- Displays tokens as they arrive in real-time
- Tracks streaming metrics including time to first token
- Sends metering data after stream completes

---

#### 4. Metadata

**File:** `openai/metadata/main.go`

Demonstrates all available metadata fields:

- Complete metadata structure
- All optional fields documented
- Subscriber information

**Run:**

```bash
make run-openai-metadata
# or
go run examples/openai/metadata/main.go
```

**What it does:**

- Shows all available metadata fields
- Demonstrates subscriber tracking
- Includes organization and product tracking

**Metadata fields supported:**

- `traceId` - Session or conversation tracking identifier
- `taskType` - Type of AI task being performed
- `agent` - AI agent or bot identifier
- `organizationId` - Organization identifier
- `productId` - Product or service identifier
- `subscriptionId` - Subscription tier identifier
- `responseQualityScore` - Quality rating (0.0-1.0)
- `subscriber` - Nested subscriber object with `id`, `email`, `credential` (with `name` and `value`)

---

### Azure OpenAI Examples

#### 1. Getting Started

**File:** `azure/getting-started/main.go`

The simplest Azure OpenAI example:

- Initialize the middleware with Azure OpenAI
- Create a basic chat completion request
- Display response and usage metrics

**Run:**

```bash
make run-azure-getting-started
# or
go run examples/azure/getting-started/main.go
```

**What it does:**

- Loads Azure OpenAI configuration from environment variables
- Creates a simple chat completion request using Azure OpenAI
- Automatically sends metering data to Revenium API
- Displays the response and token usage

---

#### 2. Basic Usage

**File:** `azure/basic/main.go`

Demonstrates standard Azure OpenAI usage:

- Chat completion with metadata
- Simple metadata tracking

**Run:**

```bash
make run-azure-basic
# or
go run examples/azure/basic/main.go
```

**What it does:**

- Creates chat completion with metadata tracking
- Demonstrates basic metadata usage with Azure OpenAI
- Shows token counting and usage metrics

---

#### 3. Streaming

**File:** `azure/streaming/main.go`

Demonstrates Azure OpenAI streaming responses:

- Real-time token streaming
- Accumulating responses
- Streaming metrics

**Run:**

```bash
make run-azure-streaming
# or
go run examples/azure/streaming/main.go
```

**What it does:**

- Creates a streaming chat completion request with Azure OpenAI
- Displays tokens as they arrive in real-time
- Tracks streaming metrics including time to first token
- Sends metering data after stream completes

---

#### 4. Metadata

**File:** `azure/metadata/main.go`

Demonstrates all available metadata fields with Azure OpenAI:

- Complete metadata structure
- All optional fields documented
- Subscriber information

**Run:**

```bash
make run-azure-metadata
# or
go run examples/azure/metadata/main.go
```

**What it does:**

- Shows all available metadata fields with Azure OpenAI
- Demonstrates subscriber tracking
- Includes organization and product tracking

---

## Common Issues

### "Client not initialized" error

**Solution:** Make sure to call `Initialize()` before using `GetClient()`.

### "REVENIUM_METERING_API_KEY is required" error

**Solution:** Set the `REVENIUM_METERING_API_KEY` environment variable in your `.env` file.

### "OPENAI_API_KEY is required" error (OpenAI)

**Solution:** Set the `OPENAI_API_KEY` environment variable in your `.env` file for OpenAI examples.

### "AZURE_OPENAI_API_KEY is required" error (Azure OpenAI)

**Solution:** Set the `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, and `AZURE_OPENAI_API_VERSION` environment variables in your `.env` file for Azure OpenAI examples.

### Environment variables not loading

**Solution:** Make sure your `.env` file is in the project root directory and contains the required variables.

### OpenAI API errors

**Solution:**

- For OpenAI: Make sure you have set `OPENAI_API_KEY` in your `.env` file
- For Azure OpenAI: Make sure you have set `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, and `AZURE_OPENAI_API_VERSION` in your `.env` file

### Debug Mode

Enable detailed logging to troubleshoot issues:

```bash
# In .env file
REVENIUM_DEBUG=true

# Then run examples
make run-openai-getting-started
```

## Next Steps

- Check the [main README](../README.md) for detailed documentation
- Visit the [Revenium Dashboard](https://app.revenium.ai) to view your metering data
- See [.env.example](../.env.example) for all configuration options

## Support

For issues or questions:

- Documentation: https://docs.revenium.io
- Email: support@revenium.io
