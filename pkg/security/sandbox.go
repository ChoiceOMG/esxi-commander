package security

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/r11/esxi-commander/pkg/metrics"
)

type OperationMode string

const (
	ModeRestricted   OperationMode = "restricted"   // Read-only + dry-run
	ModeStandard     OperationMode = "standard"     // Normal operations
	ModeUnrestricted OperationMode = "unrestricted" // Full access
)

type Sandbox struct {
	Mode           OperationMode
	AllowedOps     map[string]bool
	DryRun         bool
	PromotionToken string
	ExpiresAt      time.Time
	mu             sync.RWMutex
}

type PromotionToken struct {
	Token     string
	ExpiresAt time.Time
	FromMode  OperationMode
	ToMode    OperationMode
}

var (
	defaultSandbox *Sandbox
	sandboxMu      sync.RWMutex
)

// Restricted mode operations (read-only)
var restrictedOps = map[string]bool{
	"vm.list":     true,
	"vm.info":     true,
	"backup.list": true,
	"template.list": true,
	"datastore.list": true,
	"network.list": true,
}

// Standard mode operations (exclude destructive bulk operations)
var standardOps = map[string]bool{
	"vm.list":     true,
	"vm.info":     true,
	"vm.create":   true,
	"vm.clone":    true,
	"vm.delete":   true,
	"vm.power":    true,
	"backup.create": true,
	"backup.restore": true,
	"backup.list": true,
	"backup.delete": true,
	"template.list": true,
	"datastore.list": true,
	"network.list": true,
}

// Initialize sets up the default sandbox
func Initialize(mode OperationMode) {
	sandboxMu.Lock()
	defer sandboxMu.Unlock()
	
	defaultSandbox = &Sandbox{
		Mode:       mode,
		DryRun:     mode == ModeRestricted,
		AllowedOps: getOpsForMode(mode),
	}
}

// GetSandbox returns the current sandbox instance
func GetSandbox() *Sandbox {
	sandboxMu.RLock()
	defer sandboxMu.RUnlock()
	
	if defaultSandbox == nil {
		// Default to standard mode if not initialized
		return &Sandbox{
			Mode:       ModeStandard,
			DryRun:     false,
			AllowedOps: standardOps,
		}
	}
	return defaultSandbox
}

// CheckOperation verifies if an operation is allowed
func (s *Sandbox) CheckOperation(op string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Check if promotion has expired
	if s.PromotionToken != "" && time.Now().After(s.ExpiresAt) {
		// Revert to original mode
		s.revertPromotion()
		return fmt.Errorf("promotion expired, operation %s denied in %s mode", op, s.Mode)
	}
	
	// Record the operation attempt
	metrics.RecordAIOperation(string(s.Mode), op)
	
	// Unrestricted mode allows everything
	if s.Mode == ModeUnrestricted {
		return nil
	}
	
	// Check if operation is allowed
	if allowed, exists := s.AllowedOps[op]; !exists || !allowed {
		return fmt.Errorf("operation %s not allowed in %s mode", op, s.Mode)
	}
	
	return nil
}

// PromoteTemporary temporarily elevates privileges
func (s *Sandbox) PromoteTemporary(toMode OperationMode, duration time.Duration) (*PromotionToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Cannot promote from unrestricted
	if s.Mode == ModeUnrestricted {
		return nil, fmt.Errorf("already in unrestricted mode")
	}
	
	// Cannot demote
	if toMode == ModeRestricted && s.Mode != ModeRestricted {
		return nil, fmt.Errorf("cannot demote from %s to %s", s.Mode, toMode)
	}
	
	// Maximum promotion duration is 1 hour
	if duration > time.Hour {
		duration = time.Hour
	}
	
	token := &PromotionToken{
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(duration),
		FromMode:  s.Mode,
		ToMode:    toMode,
	}
	
	// Store original mode
	originalMode := s.Mode
	
	// Apply promotion
	s.Mode = toMode
	s.AllowedOps = getOpsForMode(toMode)
	s.DryRun = toMode == ModeRestricted
	s.PromotionToken = token.Token
	s.ExpiresAt = token.ExpiresAt
	
	// Record promotion
	metrics.RecordAIPromotion(string(originalMode), string(toMode))
	
	return token, nil
}

// EnforceRestrictions applies the sandbox restrictions
func (s *Sandbox) EnforceRestrictions() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Check if promotion has expired
	if s.PromotionToken != "" && time.Now().After(s.ExpiresAt) {
		s.revertPromotion()
	}
	
	return nil
}

// IsDryRun returns whether operations should be simulated only
func (s *Sandbox) IsDryRun() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DryRun
}

// GetMode returns the current operation mode
func (s *Sandbox) GetMode() OperationMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Mode
}

// IsOperationAllowed checks if an operation is allowed without recording metrics
func (s *Sandbox) IsOperationAllowed(op string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.Mode == ModeUnrestricted {
		return true
	}
	
	allowed, exists := s.AllowedOps[op]
	return exists && allowed
}

// revertPromotion reverts to the original mode after promotion expires
func (s *Sandbox) revertPromotion() {
	// Determine original mode based on operation set
	// This is a simplified approach - in production you'd store the original mode
	originalMode := ModeStandard
	if len(s.AllowedOps) <= len(restrictedOps) {
		originalMode = ModeRestricted
	}
	
	s.Mode = originalMode
	s.AllowedOps = getOpsForMode(originalMode)
	s.DryRun = originalMode == ModeRestricted
	s.PromotionToken = ""
	s.ExpiresAt = time.Time{}
}

// getOpsForMode returns the allowed operations for a given mode
func getOpsForMode(mode OperationMode) map[string]bool {
	switch mode {
	case ModeRestricted:
		return copyOps(restrictedOps)
	case ModeStandard:
		return copyOps(standardOps)
	case ModeUnrestricted:
		// Return nil for unrestricted (all operations allowed)
		return nil
	default:
		return copyOps(standardOps)
	}
}

// copyOps creates a copy of the operations map
func copyOps(ops map[string]bool) map[string]bool {
	result := make(map[string]bool, len(ops))
	for k, v := range ops {
		result[k] = v
	}
	return result
}

// ValidateCommand checks if a CLI command is allowed
func ValidateCommand(args []string) error {
	if len(args) < 2 {
		return nil // No operation to validate
	}
	
	// Convert command to operation
	op := commandToOperation(args)
	if op == "" {
		return nil // Unknown command, allow it
	}
	
	sandbox := GetSandbox()
	return sandbox.CheckOperation(op)
}

// commandToOperation converts CLI arguments to an operation identifier
func commandToOperation(args []string) string {
	if len(args) < 2 {
		return ""
	}
	
	// Build operation from command structure
	// e.g., ["vm", "create"] -> "vm.create"
	if args[0] == "vm" {
		switch args[1] {
		case "list":
			return "vm.list"
		case "info":
			return "vm.info"
		case "create":
			return "vm.create"
		case "clone":
			return "vm.clone"
		case "delete":
			return "vm.delete"
		case "power", "poweron", "poweroff":
			return "vm.power"
		}
	} else if args[0] == "backup" {
		switch args[1] {
		case "list":
			return "backup.list"
		case "create":
			return "backup.create"
		case "restore":
			return "backup.restore"
		case "delete":
			return "backup.delete"
		}
	}
	
	// Default operation format
	return strings.Join(args[:2], ".")
}