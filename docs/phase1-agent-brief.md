# Phase 1 Implementation Brief for ESXi Commander

## Mission
Initialize and set up the foundation for ESXi Commander - a production-grade, Ubuntu-focused VMware ESXi orchestrator. This document provides everything needed to complete Phase 1 (Week 1) of the implementation.

## Context Documents
- **Kick-off Plan**: `docs/kick-off-plan.md` - Full architecture and 6-week timeline
- **Requirements**: `REQUIREMENTS.md` - Core requirements and success metrics
- **Implementation Plan**: `docs/implementation-plan.md` - Detailed task breakdown
- **Agent Instructions**: `CLAUDE.md` - Development guidelines and conventions

## Phase 1 Objectives
Create the foundational Go project structure with CLI framework, core dependencies, and CI/CD pipeline. By the end of Phase 1, we should have:
1. A working Go project with proper module structure
2. Basic CLI that can be built and run
3. GitHub Actions CI/CD pipeline
4. First working command (`ceso vm list`)
5. Template validator foundation

## Specific Tasks

### Task 1: Initialize Go Project
```bash
# Initialize module
go mod init github.com/r11/esxi-commander

# Add core dependencies
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/rs/zerolog@latest
go get github.com/vmware/govmomi@latest
go get go.etcd.io/bbolt@latest
go get github.com/prometheus/client_golang@latest
```

### Task 2: Create Directory Structure
Create the following directory structure:
```
esxi-commander/
├── cmd/
│   ├── ceso/
│   │   └── main.go           # CLI entry point
│   └── cesod/
│       └── main.go           # Daemon entry point (stub for now)
├── pkg/
│   ├── cli/
│   │   ├── root.go          # Root command setup
│   │   ├── vm/
│   │   │   ├── vm.go        # VM subcommand
│   │   │   ├── list.go      # vm list command
│   │   │   ├── create.go    # vm create (stub)
│   │   │   └── types.go     # VM types/structs
│   │   ├── backup/
│   │   │   └── backup.go    # Backup subcommand (stub)
│   │   └── template/
│   │       ├── template.go   # Template subcommand
│   │       └── validate.go   # Template validator
│   ├── esxi/
│   │   ├── client/
│   │   │   └── client.go    # ESXi connection management
│   │   └── vm/
│   │       └── operations.go # VM operations
│   ├── cloudinit/
│   │   └── builder.go       # Cloud-init builder (stub)
│   ├── config/
│   │   └── config.go        # Configuration management
│   └── logger/
│       └── logger.go        # Zerolog setup
├── internal/
│   ├── defaults/
│   │   └── defaults.go      # Smart defaults
│   └── validation/
│       └── validation.go    # Input validators
├── test/
│   └── unit/
│       └── cli_test.go      # Basic CLI tests
├── configs/
│   └── templates.yaml       # Template definitions
└── .github/
    └── workflows/
        ├── ci.yml           # CI pipeline
        └── chaos-tests.yml  # Chaos tests (scheduled)
```

### Task 3: Implement Basic CLI Structure

**Root Command** (`pkg/cli/root.go`):
```go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    jsonOutput bool
    dryRun bool
)

var rootCmd = &cobra.Command{
    Use:   "ceso",
    Short: "ESXi Commander - Ubuntu-focused VMware ESXi orchestrator",
    Long: `ESXi Commander (ceso) is a production-grade tool for managing
Ubuntu LTS virtual machines on standalone ESXi hosts.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ceso/config.yaml)")
    rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
    rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
}
```

**Main Entry** (`cmd/ceso/main.go`):
```go
package main

import (
    "os"
    "github.com/r11/esxi-commander/pkg/cli"
    "github.com/r11/esxi-commander/pkg/logger"
)

func main() {
    logger.Init()
    
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Task 4: Implement First Command - `vm list`

Create a working `ceso vm list` command that:
1. Accepts `--json` flag for JSON output
2. For now, returns mock data (we'll connect to ESXi in Phase 2)
3. Implements proper command structure

**Expected behavior**:
```bash
$ ceso vm list
NAME           STATUS    IP              CPU    RAM(GB)
ubuntu-web-01  running   192.168.1.100   2      4
ubuntu-db-01   running   192.168.1.101   4      8
ubuntu-test    stopped   -               1      2

$ ceso vm list --json
[
  {"name": "ubuntu-web-01", "status": "running", "ip": "192.168.1.100", "cpu": 2, "ram": 4},
  {"name": "ubuntu-db-01", "status": "running", "ip": "192.168.1.101", "cpu": 4, "ram": 8},
  {"name": "ubuntu-test", "status": "stopped", "ip": "", "cpu": 1, "ram": 2}
]
```

### Task 5: Template Validator Foundation

Create `pkg/cli/template/validate.go` with basic structure:
```go
package template

type TemplateValidator struct {
    Name string
    Path string
}

type ValidationResult struct {
    Valid bool
    CloudInitInstalled bool
    VMwareToolsInstalled bool
    GuestinfoEnabled bool
    Errors []string
}

func (tv *TemplateValidator) Validate() (*ValidationResult, error) {
    // Stub implementation for Phase 1
    // Real validation will be added in Phase 2
    return &ValidationResult{
        Valid: true,
        CloudInitInstalled: true,
        VMwareToolsInstalled: true,
        GuestinfoEnabled: true,
    }, nil
}
```

### Task 6: GitHub Actions CI/CD

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      run: go test -v ./...
    
    - name: Build
      run: go build -v ./cmd/ceso
    
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Build binary
      run: |
        CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ceso ./cmd/ceso
        ./ceso version
```

### Task 7: Configuration Management

Create basic configuration structure using Viper (`pkg/config/config.go`):
```go
package config

type Config struct {
    ESXi     ESXiConfig     `yaml:"esxi"`
    Defaults DefaultsConfig `yaml:"defaults"`
    Security SecurityConfig `yaml:"security"`
}

type ESXiConfig struct {
    Host    string `yaml:"host"`
    User    string `yaml:"user"`
    SSHKey  string `yaml:"ssh_key"`
}

type DefaultsConfig struct {
    Template   string `yaml:"template"`
    Datastore  string `yaml:"datastore"`
    Network    string `yaml:"network"`
    CPU        int    `yaml:"cpu"`
    RAM        int    `yaml:"ram"`
    Disk       int    `yaml:"disk"`
}

type SecurityConfig struct {
    Mode string `yaml:"mode"` // restricted, standard, unrestricted
}
```

## Success Criteria for Phase 1

1. ✅ Go project initialized with proper module structure
2. ✅ All directories created according to specification
3. ✅ Basic CLI framework working with Cobra
4. ✅ `ceso vm list` command returns mock data
5. ✅ `--json` flag works for JSON output
6. ✅ GitHub Actions CI pipeline runs on push/PR
7. ✅ Basic tests pass
8. ✅ Project builds without errors
9. ✅ Template validator structure in place
10. ✅ Configuration management foundation ready

## Testing Phase 1

Run these commands to verify Phase 1 completion:
```bash
# Build the project
go build ./cmd/ceso

# Run the CLI
./ceso --help
./ceso vm --help
./ceso vm list
./ceso vm list --json

# Run tests
go test ./...

# Check formatting
go fmt ./...
gofmt -l .

# Verify CI passes
# Push to GitHub and check Actions tab
```

## Common Issues & Solutions

1. **Module dependencies**: If `go get` fails, ensure you're in the project root with `go.mod`
2. **Import paths**: Use full module path `github.com/r11/esxi-commander/pkg/...`
3. **Cobra commands**: Remember to add subcommands to parent in `init()` functions
4. **JSON output**: Use `encoding/json` with proper struct tags

## Deliverables Checklist

- [ ] `go.mod` and `go.sum` files
- [ ] Complete directory structure
- [ ] Working `ceso` binary
- [ ] `ceso vm list` command with JSON support
- [ ] GitHub Actions workflow files
- [ ] At least one passing unit test
- [ ] Clean `go fmt` and lint checks

## Next Steps (Phase 2 Preview)

Once Phase 1 is complete, Phase 2 will focus on:
- Connecting to real ESXi host via govmomi
- Implementing cloud-init guestinfo injection
- Creating actual VMs from templates
- Adding remaining VM commands (create, clone, delete)

## Feedback Points

Please provide updates on:
1. Any blockers or unclear requirements
2. Deviations from the plan and reasoning
3. Test results and coverage percentage
4. Build time and binary size
5. Any discovered dependencies not listed

## Important Notes

- **DO NOT** implement actual ESXi connections in Phase 1 (use mocks)
- **DO NOT** add features beyond Phase 1 scope
- **FOCUS** on clean, testable code structure
- **ENSURE** all code follows Go best practices
- **MAINTAIN** clear separation between packages
- **DOCUMENT** any assumptions made

This brief provides everything needed to complete Phase 1. Refer to the other documentation files for additional context, but this document should be your primary guide for Phase 1 implementation.