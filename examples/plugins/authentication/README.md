# Authentication Plugin

The `authentication` plugin provides a simple API key-based authentication middleware. It checks for a specific API key in the `X-API-Key` header of incoming requests.

## Configuration

The plugin is registered with the name `custom-auth`. It does not currently take any configurable options through the `cfg` map, as the API key is hardcoded within the plugin.
