package plugins

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/0xReLogic/Helios/internal/config"
)

// Middleware represents an HTTP middleware that wraps a handler
// The returned handler should call the next handler to continue the chain.
type Middleware func(next http.Handler) http.Handler

// factory constructs a middleware from a plugin name and its config payload
// The name is the configured plugin name; cfg holds arbitrary settings.
type factory func(name string, cfg map[string]interface{}) (Middleware, error)

// builtins holds registered built-in plugin factories by name
var builtins = map[string]factory{}

// RegisterBuiltin registers a built-in plugin factory
func RegisterBuiltin(name string, f factory) {
	if name == "" || f == nil {
		return
	}
	builtins[name] = f
}

// BuildChain builds the middleware chain from configuration and applies it to base.
// Order: plugins are applied in the order listed; the first plugin wraps the entire chain.
func BuildChain(pc config.PluginsConfig, base http.Handler) (http.Handler, error) {
	if base == nil {
		return nil, errors.New("base handler is nil")
	}
	if !pc.Enabled || len(pc.Chain) == 0 {
		return base, nil
	}

	h := base
	// Apply in reverse so the first listed becomes the outermost wrapper
	for i := len(pc.Chain) - 1; i >= 0; i-- {
		p := pc.Chain[i]
		f, ok := builtins[p.Name]
		if !ok {
			return nil, fmt.Errorf("unknown plugin: %s", p.Name)
		}
		mw, err := f(p.Name, p.Config)
		if err != nil {
			return nil, fmt.Errorf("plugin %s init failed: %w", p.Name, err)
		}
		h = mw(h)
	}
	return h, nil
}

// List returns the names of available built-in plugins
func List() []string {
	names := make([]string, 0, len(builtins))
	for n := range builtins {
		names = append(names, n)
	}
	return names
}
