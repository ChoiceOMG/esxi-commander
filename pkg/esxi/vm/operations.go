package vm

import (
	"context"
	"fmt"
)

type Operations struct {
	client interface{} // Will be ESXi client in Phase 2
}

func NewOperations(client interface{}) *Operations {
	return &Operations{client: client}
}

func (o *Operations) ListVMs(ctx context.Context) ([]interface{}, error) {
	// Stub implementation for Phase 1
	// Real VM listing via govmomi will be implemented in Phase 2
	return nil, fmt.Errorf("VM operations not implemented in Phase 1")
}
