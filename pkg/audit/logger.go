package audit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	auditLogger     *AuditLogger
	once            sync.Once
	sensitiveRegexp = regexp.MustCompile(`(password|secret|token|key|credential|auth)(["\s:=]+)([^"\s,}]+)`)
)

type AuditLogger struct {
	logger zerolog.Logger
	file   *os.File
	mu     sync.Mutex
}

type AuditEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	Operation     string                 `json:"operation"`
	User          string                 `json:"user"`
	Mode          string                 `json:"mode,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Result        string                 `json:"result"`
	Error         string                 `json:"error,omitempty"`
	DurationMS    int64                  `json:"duration_ms,omitempty"`
	Source        string                 `json:"source,omitempty"`
}

// Initialize sets up the audit logger
func Initialize(logPath string) error {
	var err error
	once.Do(func() {
		auditLogger, err = newAuditLogger(logPath)
	})
	return err
}

// GetLogger returns the singleton audit logger instance
func GetLogger() *AuditLogger {
	if auditLogger == nil {
		// Initialize with default path if not already done
		defaultPath := "/var/log/ceso/audit.json"
		if home, err := os.UserHomeDir(); err == nil {
			defaultPath = filepath.Join(home, ".ceso", "audit.json")
		}
		Initialize(defaultPath)
	}
	return auditLogger
}

func newAuditLogger(logPath string) (*AuditLogger, error) {
	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Create zerolog logger
	logger := zerolog.New(file).With().Timestamp().Logger()

	return &AuditLogger{
		logger: logger,
		file:   file,
	}, nil
}

// LogOperation logs an operation with context
func (a *AuditLogger) LogOperation(ctx context.Context, operation string, params map[string]interface{}) *AuditContext {
	correlationID := uuid.New().String()
	if ctxID := ctx.Value("correlation_id"); ctxID != nil {
		if id, ok := ctxID.(string); ok {
			correlationID = id
		}
	}

	return &AuditContext{
		logger:        a,
		correlationID: correlationID,
		operation:     operation,
		parameters:    redactSensitive(params),
		startTime:     time.Now(),
	}
}

// AuditContext tracks the context of an audit event
type AuditContext struct {
	logger        *AuditLogger
	correlationID string
	operation     string
	parameters    map[string]interface{}
	startTime     time.Time
}

// Success logs a successful operation
func (ac *AuditContext) Success() {
	ac.complete("success", "")
}

// Failure logs a failed operation
func (ac *AuditContext) Failure(err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	ac.complete("failure", errMsg)
}

// complete finalizes the audit log entry
func (ac *AuditContext) complete(result string, errorMsg string) {
	duration := time.Since(ac.startTime).Milliseconds()
	
	event := AuditEvent{
		Timestamp:     ac.startTime,
		CorrelationID: ac.correlationID,
		Operation:     ac.operation,
		User:          getCurrentUser(),
		Mode:          getOperationMode(),
		Parameters:    ac.parameters,
		Result:        result,
		Error:         errorMsg,
		DurationMS:    duration,
		Source:        getSource(),
	}

	ac.logger.mu.Lock()
	defer ac.logger.mu.Unlock()

	ac.logger.logger.Info().
		Str("correlation_id", event.CorrelationID).
		Str("operation", event.Operation).
		Str("user", event.User).
		Str("mode", event.Mode).
		Interface("parameters", event.Parameters).
		Str("result", event.Result).
		Str("error", event.Error).
		Int64("duration_ms", event.DurationMS).
		Str("source", event.Source).
		Msg("audit")
}

// LogRaw logs a raw audit event
func (a *AuditLogger) LogRaw(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info().
		Str("correlation_id", event.CorrelationID).
		Str("operation", event.Operation).
		Str("user", event.User).
		Str("mode", event.Mode).
		Interface("parameters", event.Parameters).
		Str("result", event.Result).
		Str("error", event.Error).
		Int64("duration_ms", event.DurationMS).
		Str("source", event.Source).
		Msg("audit")
}

// Close closes the audit log file
func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// redactSensitive redacts sensitive information from parameters
func redactSensitive(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range params {
		// Check if key contains sensitive words
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "password") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "credential") {
			result[key] = "[REDACTED]"
			continue
		}

		// Check value for sensitive patterns
		switch v := value.(type) {
		case string:
			result[key] = redactString(v)
		case map[string]interface{}:
			result[key] = redactSensitive(v)
		default:
			result[key] = value
		}
	}
	return result
}

// redactString redacts sensitive patterns in a string
func redactString(s string) string {
	// Replace sensitive patterns
	return sensitiveRegexp.ReplaceAllStringFunc(s, func(match string) string {
		parts := sensitiveRegexp.FindStringSubmatch(match)
		if len(parts) > 2 {
			return parts[1] + parts[2] + "[REDACTED]"
		}
		return match
	})
}

// getCurrentUser returns the current user (from env or system)
func getCurrentUser() string {
	// Check for AI agent user
	if user := os.Getenv("CESO_USER"); user != "" {
		return user
	}
	
	// Check for system user
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	
	return "unknown"
}

// getOperationMode returns the current operation mode
func getOperationMode() string {
	// This would integrate with the security sandbox
	if mode := os.Getenv("CESO_SECURITY_MODE"); mode != "" {
		return mode
	}
	return "standard"
}

// getSource returns the source of the operation (CLI, API, etc.)
func getSource() string {
	if source := os.Getenv("CESO_SOURCE"); source != "" {
		return source
	}
	return "cli"
}

// QuickLog provides a simple way to log an operation
func QuickLog(operation string, params map[string]interface{}, result string, err error) {
	logger := GetLogger()
	ctx := logger.LogOperation(context.Background(), operation, params)
	
	if err != nil {
		ctx.Failure(err)
	} else {
		ctx.Success()
	}
}