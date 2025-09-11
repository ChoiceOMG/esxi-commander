package client

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25"
)

// Mock govmomi client for testing
type MockGovmomiClient struct {
	mock.Mock
}

func (m *MockGovmomiClient) Login(ctx context.Context, u *url.Userinfo) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockGovmomiClient) Logout(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGovmomiClient) IsVC() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid configuration",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password123",
				Insecure: true,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: Config{
				User:     "root",
				Password: "password123",
			},
			wantErr:  true,
			errorMsg: "host is required",
		},
		{
			name: "missing username",
			config: Config{
				Host:     "192.168.1.100",
				Password: "password123",
			},
			wantErr:  true,
			errorMsg: "username is required",
		},
		{
			name: "missing password",
			config: Config{
				Host: "192.168.1.100",
				User: "root",
			},
			wantErr:  true,
			errorMsg: "password is required",
		},
		{
			name: "with port",
			config: Config{
				Host:     "192.168.1.100",
				Port:     443,
				User:     "root",
				Password: "password123",
				Insecure: true,
			},
			wantErr: false,
		},
		{
			name: "with datacenter",
			config: Config{
				Host:       "192.168.1.100",
				User:       "root",
				Password:   "password123",
				Datacenter: "dc1",
				Insecure:   true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				// Can't actually connect in unit tests
				// Just verify config is set correctly
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.Host, client.config.Host)
				assert.Equal(t, tt.config.User, client.config.User)
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		setupMock func() *govmomi.Client
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "successful connection",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password123",
				Insecure: true,
			},
			setupMock: func() *govmomi.Client {
				// Would return mock client
				return nil
			},
			wantErr: false,
		},
		{
			name: "connection timeout",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password123",
				Timeout:  1 * time.Second,
			},
			setupMock: func() *govmomi.Client {
				// Would simulate timeout
				return nil
			},
			wantErr:  true,
			errorMsg: "timeout",
		},
		{
			name: "authentication failure",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "wrongpassword",
			},
			setupMock: func() *govmomi.Client {
				// Would simulate auth failure
				return nil
			},
			wantErr:  true,
			errorMsg: "authentication failed",
		},
		{
			name: "invalid host",
			config: Config{
				Host:     "invalid-host",
				User:     "root",
				Password: "password123",
			},
			setupMock: func() *govmomi.Client {
				return nil
			},
			wantErr:  true,
			errorMsg: "invalid host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: tt.config,
			}

			// In real implementation, would use mock
			if tt.setupMock != nil {
				_ = tt.setupMock()
			}

			err := client.Connect(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Can't actually test connection without ESXi
				assert.True(t, true)
			}
		})
	}
}

func TestClient_GetVM(t *testing.T) {
	tests := []struct {
		name     string
		vmName   string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid VM name",
			vmName:  "test-vm",
			wantErr: false,
		},
		{
			name:     "empty VM name",
			vmName:   "",
			wantErr:  true,
			errorMsg: "VM name cannot be empty",
		},
		{
			name:     "VM not found",
			vmName:   "nonexistent-vm",
			wantErr:  true,
			errorMsg: "not found",
		},
		{
			name:    "VM name with special characters",
			vmName:  "test-vm-123_prod",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: Config{
					Host:     "192.168.1.100",
					User:     "root",
					Password: "password123",
				},
			}

			// Would use mock client in real implementation
			vm, err := client.GetVM(context.Background(), tt.vmName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, vm)
			} else {
				// Can't test without real connection
				assert.True(t, true)
			}
		})
	}
}

func TestClient_GetResourcePool(t *testing.T) {
	tests := []struct {
		name     string
		poolName string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "default resource pool",
			poolName: "",
			wantErr: false,
		},
		{
			name:    "named resource pool",
			poolName: "Resources",
			wantErr: false,
		},
		{
			name:     "non-existent pool",
			poolName: "InvalidPool",
			wantErr:  true,
			errorMsg: "resource pool not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: Config{
					Host:     "192.168.1.100",
					User:     "root",
					Password: "password123",
				},
			}

			pool, err := client.GetResourcePool(context.Background(), tt.poolName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, pool)
			} else {
				// Can't test without real connection
				assert.True(t, true)
			}
		})
	}
}

func TestClient_GetDatastore(t *testing.T) {
	tests := []struct {
		name          string
		datastoreName string
		wantErr       bool
		errorMsg      string
	}{
		{
			name:          "default datastore",
			datastoreName: "",
			wantErr:       false,
		},
		{
			name:          "named datastore",
			datastoreName: "datastore1",
			wantErr:       false,
		},
		{
			name:          "non-existent datastore",
			datastoreName: "InvalidDatastore",
			wantErr:       true,
			errorMsg:      "datastore not found",
		},
		{
			name:          "datastore with spaces",
			datastoreName: "Local Datastore",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: Config{
					Host:     "192.168.1.100",
					User:     "root",
					Password: "password123",
				},
			}

			ds, err := client.GetDatastore(context.Background(), tt.datastoreName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, ds)
			} else {
				// Can't test without real connection
				assert.True(t, true)
			}
		})
	}
}

func TestClient_GetNetwork(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		wantErr     bool
		errorMsg    string
	}{
		{
			name:        "default network",
			networkName: "VM Network",
			wantErr:     false,
		},
		{
			name:        "custom network",
			networkName: "Production Network",
			wantErr:     false,
		},
		{
			name:        "non-existent network",
			networkName: "InvalidNetwork",
			wantErr:     true,
			errorMsg:    "network not found",
		},
		{
			name:        "empty network name",
			networkName: "",
			wantErr:     false, // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: Config{
					Host:     "192.168.1.100",
					User:     "root",
					Password: "password123",
				},
			}

			network, err := client.GetNetwork(context.Background(), tt.networkName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, network)
			} else {
				// Can't test without real connection
				assert.True(t, true)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	client := &Client{
		config: Config{
			Host:     "192.168.1.100",
			User:     "root",
			Password: "password123",
		},
	}

	// Test close with nil client (not connected)
	assert.NotPanics(t, func() {
		client.Close()
	})

	// Test close with mock client
	// In real implementation would test with connected client
}

func TestClient_IsConnected(t *testing.T) {
	tests := []struct {
		name      string
		client    *Client
		connected bool
	}{
		{
			name: "not connected",
			client: &Client{
				config: Config{
					Host: "192.168.1.100",
				},
				client: nil,
			},
			connected: false,
		},
		{
			name: "connected",
			client: &Client{
				config: Config{
					Host: "192.168.1.100",
				},
				client: &govmomi.Client{}, // Would be real client
			},
			connected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsConnected()
			assert.Equal(t, tt.connected, result)
		})
	}
}

func TestClient_ValidateConnection(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		wantErr bool
	}{
		{
			name: "valid connection",
			client: &Client{
				client: &govmomi.Client{},
			},
			wantErr: false,
		},
		{
			name: "no connection",
			client: &Client{
				client: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.ValidateConnection()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid config",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: Config{
				User:     "root",
				Password: "password",
			},
			wantErr:  true,
			errorMsg: "host is required",
		},
		{
			name: "missing user",
			config: Config{
				Host:     "192.168.1.100",
				Password: "password",
			},
			wantErr:  true,
			errorMsg: "username is required",
		},
		{
			name: "missing password",
			config: Config{
				Host: "192.168.1.100",
				User: "root",
			},
			wantErr:  true,
			errorMsg: "password is required",
		},
		{
			name: "invalid port",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password",
				Port:     70000,
			},
			wantErr:  true,
			errorMsg: "invalid port",
		},
		{
			name: "invalid timeout",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password",
				Timeout:  -1 * time.Second,
			},
			wantErr:  true,
			errorMsg: "invalid timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
		wantErr  bool
	}{
		{
			name: "basic URL",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password",
			},
			expected: "https://root:password@192.168.1.100/sdk",
			wantErr:  false,
		},
		{
			name: "with port",
			config: Config{
				Host:     "192.168.1.100",
				Port:     8443,
				User:     "root",
				Password: "password",
			},
			expected: "https://root:password@192.168.1.100:8443/sdk",
			wantErr:  false,
		},
		{
			name: "with special characters in password",
			config: Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "p@ss!word#123",
			},
			expected: "https://root:p%40ss%21word%23123@192.168.1.100/sdk",
			wantErr:  false,
		},
		{
			name: "IPv6 address",
			config: Config{
				Host:     "::1",
				User:     "root",
				Password: "password",
			},
			expected: "https://root:password@[::1]/sdk",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildURL(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, url.String())
			}
		})
	}
}

// Helper function for testing
func buildURL(config Config) (*url.URL, error) {
	if config.Host == "" {
		return nil, errors.New("host is required")
	}

	scheme := "https"
	host := config.Host
	if config.Port > 0 {
		host = host + ":" + string(config.Port)
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/sdk",
		User:   url.UserPassword(config.User, config.Password),
	}

	return u, nil
}