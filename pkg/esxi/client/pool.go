package client

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

// ConnectionPool manages a pool of ESXi client connections
type ConnectionPool struct {
	config      *Config
	connections chan *ESXiClient
	maxSize     int
	minSize     int
	mu          sync.RWMutex
	closed      bool
	stats       PoolStats
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	Created    int64
	Destroyed  int64
	Active     int64
	Idle       int64
	Timeouts   int64
	Errors     int64
}

// PoolConfig defines connection pool configuration
type PoolConfig struct {
	MinConnections int
	MaxConnections int
	MaxIdleTime    time.Duration
	HealthCheckInterval time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MinConnections: 2,
		MaxConnections: 10,
		MaxIdleTime:    5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *Config, poolConfig *PoolConfig) (*ConnectionPool, error) {
	if poolConfig == nil {
		poolConfig = DefaultPoolConfig()
	}

	pool := &ConnectionPool{
		config:      config,
		connections: make(chan *ESXiClient, poolConfig.MaxConnections),
		maxSize:     poolConfig.MaxConnections,
		minSize:     poolConfig.MinConnections,
	}

	// Pre-warm the pool with minimum connections
	for i := 0; i < poolConfig.MinConnections; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Clean up any created connections
			pool.Close()
			return nil, fmt.Errorf("failed to initialize connection pool: %w", err)
		}
		pool.connections <- conn
		pool.stats.Created++
		pool.stats.Idle++
	}

	// Start health check routine
	go pool.healthCheckLoop(poolConfig.HealthCheckInterval)

	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*ESXiClient, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	p.mu.RUnlock()

	select {
	case conn := <-p.connections:
		// Got connection from pool
		p.mu.Lock()
		p.stats.Idle--
		p.stats.Active++
		p.mu.Unlock()

		// Check if connection is still valid
		if err := p.validateConnection(ctx, conn); err != nil {
			// Connection is invalid, create a new one
			p.mu.Lock()
			p.stats.Destroyed++
			p.mu.Unlock()
			
			newConn, err := p.createConnection()
			if err != nil {
				p.mu.Lock()
				p.stats.Active--
				p.stats.Errors++
				p.mu.Unlock()
				return nil, err
			}
			
			p.mu.Lock()
			p.stats.Created++
			p.mu.Unlock()
			return newConn, nil
		}

		return conn, nil

	case <-time.After(5 * time.Second):
		// No connection available, try to create new one if under limit
		p.mu.Lock()
		if int(p.stats.Created-p.stats.Destroyed) < p.maxSize {
			p.mu.Unlock()
			conn, err := p.createConnection()
			if err != nil {
				p.mu.Lock()
				p.stats.Errors++
				p.mu.Unlock()
				return nil, err
			}
			
			p.mu.Lock()
			p.stats.Created++
			p.stats.Active++
			p.mu.Unlock()
			return conn, nil
		}
		p.stats.Timeouts++
		p.mu.Unlock()
		return nil, fmt.Errorf("connection pool timeout: no available connections")

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *ESXiClient) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		conn.Close()
		return
	}
	p.mu.RUnlock()

	p.mu.Lock()
	p.stats.Active--
	p.stats.Idle++
	p.mu.Unlock()

	select {
	case p.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
		p.mu.Lock()
		p.stats.Destroyed++
		p.stats.Idle--
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Close all connections
	close(p.connections)
	for conn := range p.connections {
		conn.Close()
		p.mu.Lock()
		p.stats.Destroyed++
		p.mu.Unlock()
	}

	return nil
}

// GetStats returns current pool statistics
func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// createConnection creates a new ESXi client connection
func (p *ConnectionPool) createConnection() (*ESXiClient, error) {
	ctx := context.Background()
	
	u, err := soap.ParseURL(fmt.Sprintf("https://%s/sdk", p.config.Host))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	u.User = url.UserPassword(p.config.User, p.config.Password)

	soapClient := soap.NewClient(u, p.config.Insecure)
	if p.config.Timeout > 0 {
		soapClient.Timeout = p.config.Timeout
	}

	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create vim client: %w", err)
	}

	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	err = client.SessionManager.Login(ctx, u.User)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Create finder and get datacenter
	finder := find.NewFinder(client.Client)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	return &ESXiClient{
		client:     client,
		datacenter: dc,
		finder:     finder,
		config:     p.config,
	}, nil
}

// validateConnection checks if a connection is still valid
func (p *ConnectionPool) validateConnection(ctx context.Context, conn *ESXiClient) error {
	// Check if session is still active
	sessionMgr := session.NewManager(conn.client.Client)
	
	userSession, err := sessionMgr.UserSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user session: %w", err)
	}

	if userSession == nil {
		return fmt.Errorf("session expired")
	}

	// Optionally perform a lightweight operation to verify connectivity
	// This is a basic ping to ensure connection is alive
	// In real implementation, would use conn.Client.ServiceContent.About

	return nil
}

// healthCheckLoop periodically checks connection health
func (p *ConnectionPool) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		}

		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()
	}
}

// performHealthCheck checks and refreshes connections as needed
func (p *ConnectionPool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check a sample of idle connections
	var connsToCheck []*ESXiClient
	
	// Get up to 3 connections to check
	for i := 0; i < 3; i++ {
		select {
		case conn := <-p.connections:
			connsToCheck = append(connsToCheck, conn)
		default:
			break
		}
	}

	// Check each connection and put back or replace
	for _, conn := range connsToCheck {
		if err := p.validateConnection(ctx, conn); err != nil {
			// Connection is bad, close it and create new one
			conn.Close()
			p.mu.Lock()
			p.stats.Destroyed++
			p.mu.Unlock()

			if newConn, err := p.createConnection(); err == nil {
				p.connections <- newConn
				p.mu.Lock()
				p.stats.Created++
				p.mu.Unlock()
			} else {
				p.mu.Lock()
				p.stats.Errors++
				p.mu.Unlock()
			}
		} else {
			// Connection is good, put it back
			p.connections <- conn
		}
	}

	// Ensure minimum connections
	p.mu.RLock()
	currentSize := int(p.stats.Created - p.stats.Destroyed)
	p.mu.RUnlock()

	for currentSize < p.minSize {
		if conn, err := p.createConnection(); err == nil {
			p.connections <- conn
			p.mu.Lock()
			p.stats.Created++
			p.stats.Idle++
			p.mu.Unlock()
			currentSize++
		} else {
			break
		}
	}
}

// WithConnection executes a function with a connection from the pool
func (p *ConnectionPool) WithConnection(ctx context.Context, fn func(*ESXiClient) error) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(conn)

	return fn(conn)
}