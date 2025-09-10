package client

import (
	"context"
	"fmt"
)

type ESXiClient struct {
	Host     string
	Username string
	Password string
}

func NewClient(host, username, password string) *ESXiClient {
	return &ESXiClient{
		Host:     host,
		Username: username,
		Password: password,
	}
}

func (c *ESXiClient) Connect(ctx context.Context) error {
	// Stub implementation for Phase 1
	// Real govmomi connection will be implemented in Phase 2
	return fmt.Errorf("ESXi connection not implemented in Phase 1")
}
