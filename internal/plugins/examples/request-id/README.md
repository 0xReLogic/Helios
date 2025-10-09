# Request ID Plugin

The `request-id` plugin generates a unique request ID for each incoming request. It creates a random 64-bit integer, converts it to a string, and sets it as the `X-Request-ID` header in both the incoming request and the outgoing response.

## Configuration

The plugin is registered with the name `request-id`. It does not take any configurable options.
