package loadbalancer

import (
	"net"
	"sync"
	"time"

	"github.com/0xReLogic/Helios/internal/logging"
)

// WebSocketPool manages a pool of WebSocket connections for connection reuse
type WebSocketPool struct {
	pools       map[string]*connPool
	mu          sync.RWMutex
	maxIdle     int
	maxActive   int
	idleTimeout time.Duration
}

// connPool holds connections for a specific backend
type connPool struct {
	backend     string
	idle        []pooledConn
	active      int
	mu          sync.Mutex
	idleTimeout time.Duration
}

// pooledConn wraps a connection with metadata
type pooledConn struct {
	conn     net.Conn
	lastUsed time.Time
	backend  string
}

// NewWebSocketPool creates a new WebSocket connection pool
func NewWebSocketPool(maxIdle, maxActive int, idleTimeout time.Duration) *WebSocketPool {
	pool := &WebSocketPool{
		pools:       make(map[string]*connPool),
		maxIdle:     maxIdle,
		maxActive:   maxActive,
		idleTimeout: idleTimeout,
	}

	// Start cleanup goroutine to remove stale connections
	go pool.cleanupLoop()

	return pool
}

// Get retrieves a connection from the pool or returns nil if none available
func (p *WebSocketPool) Get(backend string) net.Conn {
	p.mu.RLock()
	pool, exists := p.pools[backend]
	p.mu.RUnlock()

	if !exists {
		return nil
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Try to get an idle connection
	for len(pool.idle) > 0 {
		pc := pool.idle[len(pool.idle)-1]
		pool.idle = pool.idle[:len(pool.idle)-1]

		// Check if connection is still valid and not stale
		if time.Since(pc.lastUsed) > pool.idleTimeout {
			_ = pc.conn.Close() // Best effort close, ignore error
			continue
		}

		pool.active++
		return pc.conn
	}

	return nil
}

// Put returns a connection to the pool
func (p *WebSocketPool) Put(backend string, conn net.Conn) bool {
	if conn == nil {
		return false
	}

	p.mu.Lock()
	pool, exists := p.pools[backend]
	if !exists {
		pool = &connPool{
			backend:     backend,
			idle:        make([]pooledConn, 0, p.maxIdle),
			idleTimeout: p.idleTimeout,
		}
		p.pools[backend] = pool
	}
	p.mu.Unlock()

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.active > 0 {
		pool.active--
	}

	// Don't exceed max idle connections
	if len(pool.idle) >= p.maxIdle {
		_ = conn.Close()
		return false
	}

	pool.idle = append(pool.idle, pooledConn{
		conn:     conn,
		lastUsed: time.Now(),
		backend:  backend,
	})

	return true
}

// Close closes a connection and decrements active count
func (p *WebSocketPool) Close(backend string, conn net.Conn) {
	if conn != nil {
		_ = conn.Close()
	}

	p.mu.RLock()
	pool, exists := p.pools[backend]
	p.mu.RUnlock()

	if !exists {
		return
	}

	pool.mu.Lock()
	if pool.active > 0 {
		pool.active--
	}
	pool.mu.Unlock()
}

// Stats returns statistics for a backend's connection pool
func (p *WebSocketPool) Stats(backend string) (idle, active int) {
	p.mu.RLock()
	pool, exists := p.pools[backend]
	p.mu.RUnlock()

	if !exists {
		return 0, 0
	}

	pool.mu.Lock()
	idle = len(pool.idle)
	active = pool.active
	pool.mu.Unlock()

	return idle, active
}

// cleanupLoop periodically removes stale connections from all pools
func (p *WebSocketPool) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes stale connections from all backend pools
func (p *WebSocketPool) cleanup() {
	p.mu.RLock()
	backends := make([]string, 0, len(p.pools))
	for backend := range p.pools {
		backends = append(backends, backend)
	}
	p.mu.RUnlock()

	for _, backend := range backends {
		p.mu.RLock()
		pool, exists := p.pools[backend]
		p.mu.RUnlock()

		if !exists {
			continue
		}

		pool.mu.Lock()
		validConns := make([]pooledConn, 0, len(pool.idle))
		closedCount := 0

		for _, pc := range pool.idle {
			if time.Since(pc.lastUsed) > pool.idleTimeout {
				_ = pc.conn.Close() // Best effort close, ignore error
				closedCount++
			} else {
				validConns = append(validConns, pc)
			}
		}

		pool.idle = validConns
		pool.mu.Unlock()

		if closedCount > 0 {
			logging.L().Debug().
				Str("backend", backend).
				Int("closed", closedCount).
				Int("remaining", len(validConns)).
				Msg("cleaned up stale WebSocket connections")
		}
	}
}

// Shutdown closes all connections in all pools
func (p *WebSocketPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for backend, pool := range p.pools {
		pool.mu.Lock()
		for _, pc := range pool.idle {
			_ = pc.conn.Close() // Best effort close, ignore error
		}
		pool.idle = nil
		pool.mu.Unlock()

		logging.L().Info().
			Str("backend", backend).
			Msg("closed all WebSocket connections for backend")
	}

	p.pools = make(map[string]*connPool)
}
