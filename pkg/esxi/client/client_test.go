package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "valid config",
			config: &Config{
				Host:     "192.168.1.100",
				User:     "root",
				Password: "password",
				Insecure: true,
				Timeout:  30 * time.Second,
			},
			valid: true,
		},
		{
			name: "missing host",
			config: &Config{
				User:     "root",
				Password: "password",
			},
			valid: false,
		},
		{
			name: "missing credentials",
			config: &Config{
				Host: "192.168.1.100",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.config.Host)
				assert.NotEmpty(t, tt.config.User)
				assert.NotEmpty(t, tt.config.Password)
			} else {
				// Check that at least one required field is missing
				assert.True(t, tt.config.Host == "" || tt.config.User == "" || tt.config.Password == "")
			}
		})
	}
}

func TestVMStruct(t *testing.T) {
	vm := &VM{
		Name:   "test-vm",
		UUID:   "uuid-123",
		Status: "poweredOn",
		IP:     "192.168.1.100",
		CPU:    2,
		Memory: 4,
	}

	assert.Equal(t, "test-vm", vm.Name)
	assert.Equal(t, "uuid-123", vm.UUID)
	assert.Equal(t, "poweredOn", vm.Status)
	assert.Equal(t, "192.168.1.100", vm.IP)
	assert.Equal(t, 2, vm.CPU)
	assert.Equal(t, 4, vm.Memory)
}

func TestConnectionPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 2, config.MinConnections)
	assert.Equal(t, 10, config.MaxConnections)
	assert.Equal(t, 5*time.Minute, config.MaxIdleTime)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
}

func TestPoolStats(t *testing.T) {
	stats := PoolStats{
		Created:   10,
		Destroyed: 2,
		Active:    3,
		Idle:      5,
		Timeouts:  1,
		Errors:    0,
	}

	assert.Equal(t, int64(10), stats.Created)
	assert.Equal(t, int64(2), stats.Destroyed)
	assert.Equal(t, int64(3), stats.Active)
	assert.Equal(t, int64(5), stats.Idle)
	assert.Equal(t, int64(1), stats.Timeouts)
	assert.Equal(t, int64(0), stats.Errors)
}