package loadbalancer

import (
	"net"
	"testing"
	"time"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	closed bool
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestWebSocketPool_GetPut(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"
	conn := &mockConn{}

	// Initially, Get should return nil (no connections in pool)
	retrieved := pool.Get(backend)
	if retrieved != nil {
		t.Error("Expected nil when pool is empty")
	}

	// Put a connection
	if !pool.Put(backend, conn) {
		t.Error("Failed to put connection in pool")
	}

	// Now Get should return the connection
	retrieved = pool.Get(backend)
	if retrieved != conn {
		t.Error("Expected to get the same connection back")
	}

	// Getting again should return nil (connection already retrieved)
	retrieved = pool.Get(backend)
	if retrieved != nil {
		t.Error("Expected nil after connection was already retrieved")
	}
}

func TestWebSocketPool_MaxIdle(t *testing.T) {
	maxIdle := 3
	pool := NewWebSocketPool(maxIdle, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"

	// Add more connections than maxIdle
	conns := make([]*mockConn, 5)
	for i := 0; i < 5; i++ {
		conns[i] = &mockConn{}
		pool.Put(backend, conns[i])
	}

	// Only maxIdle connections should be in the pool
	idle, _ := pool.Stats(backend)
	if idle != maxIdle {
		t.Errorf("Expected %d idle connections, got %d", maxIdle, idle)
	}

	// Excess connections should be closed
	closedCount := 0
	for _, conn := range conns {
		if conn.closed {
			closedCount++
		}
	}
	if closedCount != 2 {
		t.Errorf("Expected 2 connections to be closed, got %d", closedCount)
	}
}

func TestWebSocketPool_IdleTimeout(t *testing.T) {
	idleTimeout := 100 * time.Millisecond
	pool := NewWebSocketPool(5, 10, idleTimeout)
	defer pool.Shutdown()

	backend := "backend1"
	conn := &mockConn{}

	// Put a connection
	pool.Put(backend, conn)

	// Wait for connection to become stale
	time.Sleep(idleTimeout + 50*time.Millisecond)

	// Try to get the connection - should be nil because it's stale
	retrieved := pool.Get(backend)
	if retrieved != nil {
		t.Error("Expected nil for stale connection")
	}

	// Connection should have been closed
	if !conn.closed {
		t.Error("Expected stale connection to be closed")
	}
}

func TestWebSocketPool_Stats(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"

	// Initially, stats should be zero
	idle, active := pool.Stats(backend)
	if idle != 0 || active != 0 {
		t.Errorf("Expected 0/0, got %d/%d", idle, active)
	}

	// Add some connections
	conn1 := &mockConn{}
	conn2 := &mockConn{}
	pool.Put(backend, conn1)
	pool.Put(backend, conn2)

	idle, active = pool.Stats(backend)
	if idle != 2 || active != 0 {
		t.Errorf("Expected 2/0, got %d/%d", idle, active)
	}

	// Get a connection
	pool.Get(backend)

	idle, active = pool.Stats(backend)
	if idle != 1 || active != 1 {
		t.Errorf("Expected 1/1, got %d/%d", idle, active)
	}
}

func TestWebSocketPool_Close(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"
	conn := &mockConn{}

	// Get a connection (simulating active)
	pool.Put(backend, conn)
	pool.Get(backend)

	_, active := pool.Stats(backend)
	if active != 1 {
		t.Errorf("Expected 1 active connection, got %d", active)
	}

	// Close the connection
	pool.Close(backend, conn)

	_, active = pool.Stats(backend)
	if active != 0 {
		t.Errorf("Expected 0 active connections after close, got %d", active)
	}

	if !conn.closed {
		t.Error("Expected connection to be closed")
	}
}

func TestWebSocketPool_Cleanup(t *testing.T) {
	idleTimeout := 200 * time.Millisecond
	pool := NewWebSocketPool(5, 10, idleTimeout)
	defer pool.Shutdown()

	backend := "backend1"
	
	// Add some connections
	conns := make([]*mockConn, 3)
	for i := 0; i < 3; i++ {
		conns[i] = &mockConn{}
		pool.Put(backend, conns[i])
	}

	idle, _ := pool.Stats(backend)
	if idle != 3 {
		t.Errorf("Expected 3 idle connections, got %d", idle)
	}

	// Wait for connections to become stale
	time.Sleep(idleTimeout + 100*time.Millisecond)

	// Run cleanup manually
	pool.cleanup()

	idle, _ = pool.Stats(backend)
	if idle != 0 {
		t.Errorf("Expected 0 idle connections after cleanup, got %d", idle)
	}

	// All connections should be closed
	for i, conn := range conns {
		if !conn.closed {
			t.Errorf("Expected connection %d to be closed", i)
		}
	}
}

func TestWebSocketPool_MultipleBackends(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend1 := "backend1"
	backend2 := "backend2"

	conn1 := &mockConn{}
	conn2 := &mockConn{}

	// Put connections for different backends
	pool.Put(backend1, conn1)
	pool.Put(backend2, conn2)

	// Get from specific backend
	retrieved1 := pool.Get(backend1)
	if retrieved1 != conn1 {
		t.Error("Expected to get connection from backend1")
	}

	retrieved2 := pool.Get(backend2)
	if retrieved2 != conn2 {
		t.Error("Expected to get connection from backend2")
	}

	// Stats should be independent
	idle1, active1 := pool.Stats(backend1)
	idle2, active2 := pool.Stats(backend2)

	if idle1 != 0 || active1 != 1 {
		t.Errorf("Backend1: expected 0/1, got %d/%d", idle1, active1)
	}
	if idle2 != 0 || active2 != 1 {
		t.Errorf("Backend2: expected 0/1, got %d/%d", idle2, active2)
	}
}

func TestWebSocketPool_Shutdown(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)

	backend := "backend1"
	conns := make([]*mockConn, 3)
	
	for i := 0; i < 3; i++ {
		conns[i] = &mockConn{}
		pool.Put(backend, conns[i])
	}

	// Shutdown should close all connections
	pool.Shutdown()

	for i, conn := range conns {
		if !conn.closed {
			t.Errorf("Expected connection %d to be closed after shutdown", i)
		}
	}

	// Stats should be zero after shutdown
	idle, active := pool.Stats(backend)
	if idle != 0 || active != 0 {
		t.Errorf("Expected 0/0 after shutdown, got %d/%d", idle, active)
	}
}

func TestWebSocketPool_NilConnection(t *testing.T) {
	pool := NewWebSocketPool(5, 10, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"

	// Putting nil should return false
	if pool.Put(backend, nil) {
		t.Error("Expected Put to return false for nil connection")
	}

	// Closing nil should not panic
	pool.Close(backend, nil)
}

func TestWebSocketPool_ConcurrentAccess(t *testing.T) {
	pool := NewWebSocketPool(10, 20, 1*time.Minute)
	defer pool.Shutdown()

	backend := "backend1"
	done := make(chan bool)

	// Concurrent puts
	go func() {
		for i := 0; i < 100; i++ {
			conn := &mockConn{}
			pool.Put(backend, conn)
		}
		done <- true
	}()

	// Concurrent gets
	go func() {
		for i := 0; i < 100; i++ {
			conn := pool.Get(backend)
			if conn != nil {
				pool.Put(backend, conn)
			}
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Should not panic and should have valid stats
	idle, active := pool.Stats(backend)
	if idle < 0 || active < 0 {
		t.Errorf("Invalid stats after concurrent access: %d/%d", idle, active)
	}
}
