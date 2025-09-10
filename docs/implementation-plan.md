# ESXi Commander Implementation Plan

Based on the kick-off plan review, here's the complete implementation roadmap:

## Phase 1: Project Foundation (Week 1)

### 1. Initialize Go project
- Create `go.mod` with module `github.com/r11/esxi-commander`
- Set up project directory structure

### 2. Create project structure
- `cmd/ceso/` - Main CLI binary
- `cmd/cesod/` - Optional JSON-RPC daemon
- `pkg/` - Core packages (cli, esxi, cloudinit, backup, security, audit)
- `internal/` - Internal utilities (defaults, validation, storage)
- `test/` - Test suites (unit, integration, chaos)
- `configs/` - Configuration templates

### 3. Set up CI/CD pipeline
- GitHub Actions workflows for testing, building, and chaos tests
- Acceptance criteria gates (performance metrics, success rates)

### 4. Implement basic CLI with Cobra
- Initialize Cobra framework with root command
- Add VM and backup subcommands structure
- Implement `--json` and `--dry-run` global flags

### 5. Configure dependencies
- govmomi for VMware SDK
- zerolog for structured logging
- BoltDB for embedded storage
- Prometheus client for metrics

## Phase 2: Core VM Operations (Week 2)

### 6. Template validator
- Validate Ubuntu templates for cloud-init readiness
- Check VMware Tools installation
- Verify guestinfo datasource configuration

### 7. Cloud-init guestinfo builder
- Generate metadata and userdata
- Implement gzip+base64 encoding
- Support static IP, hostname, SSH key injection

### 8. ESXi adapter layer
- govmomi client connection management
- SSH fallback implementation
- Command execution abstraction

### 9. VM operations
- `vm list` - List all VMs with status
- `vm create` - Create from template with cloud-init
- `vm clone` - Cold clone with re-IP
- `vm info` - Detailed VM information
- `vm delete` - Safe VM deletion

## Phase 3: Testing Infrastructure (Week 3)

### 10. Unit test framework
- Command builders tests
- Cloud-init generators tests
- Input validators tests
- Target: 70% coverage

### 11. Integration tests
- VM lifecycle testing
- Cloud-init verification
- Network configuration validation

### 12. Chaos testing
- Datastore full scenarios
- Network partition handling
- Concurrent operation safety

## Phase 4: Backup System (Week 4)

### 13. Backup catalog schema
- BoltDB schema for backup metadata
- Forward-compatible versioning
- Retention policy support

### 14. Backup operations
- `backup create` - Cold backup with compression
- `backup restore` - Restore with re-IP
- `backup list` - List available backups

### 15. Extensible target interface
- Abstract backup target interface
- Datastore target implementation
- Future-proof for S3/NFS/SFTP

## Phase 5: Security & Monitoring (Week 5)

### 16. AI agent sandboxing
- Restricted mode for AI agents
- Promotion workflow with time limits
- Operation mode enforcement

### 17. Security implementation
- IP allowlisting
- Secret management (keyring/encrypted)
- Audit log redaction

### 18. Prometheus metrics
- Operation duration histograms
- Success/failure counters
- AI agent activity tracking

### 19. Grafana dashboards
- Operational metrics dashboard
- AI agent activity dashboard

## Phase 6: Production Hardening (Week 6)

### 20. Performance optimization
- Connection pooling
- Parallel operation support
- Resource usage optimization

### 21. Documentation
- CLI usage documentation
- AI agent integration guide
- Configuration examples

### 22. Additional chaos tests
- Snapshot failure recovery
- API timeout handling
- Resource exhaustion scenarios

## Key Implementation Details

### Configuration Management
- YAML configuration with Viper
- Smart defaults for common operations
- Template definitions and validation status

### Success Criteria
- VM create: <90s
- Clone 80GB: <5min
- Backup success: >99%
- Zero manual ESXi UI steps
- Complete audit trail

### Technology Stack
- Go for single binary deployment
- govmomi for ESXi API (primary)
- SSH fallback for vim-cmd/esxcli
- BoltDB for embedded storage
- Prometheus + Grafana for monitoring

## Task Tracking

### Week 1 Tasks
- [ ] Initialize Go project with module structure
- [ ] Create directory structure (cmd/, pkg/, internal/, test/)
- [ ] Set up GitHub Actions CI/CD
- [ ] Implement basic Cobra CLI framework
- [ ] Add template validator implementation
- [ ] Configure core dependencies

### Week 2 Tasks
- [ ] Build cloud-init guestinfo injection system
- [ ] Create ESXi adapter with govmomi
- [ ] Implement VM create command
- [ ] Add VM list and info commands
- [ ] Implement VM clone functionality
- [ ] Add VM delete command

### Week 3 Tasks
- [ ] Set up unit test framework with testify
- [ ] Write unit tests for core components
- [ ] Create integration test suite
- [ ] Add first chaos test
- [ ] Configure test coverage reporting

### Week 4 Tasks
- [ ] Design backup catalog schema
- [ ] Implement backup create operation
- [ ] Add backup restore functionality
- [ ] Create backup list command
- [ ] Build extensible target interface

### Week 5 Tasks
- [ ] Implement AI agent sandboxing
- [ ] Add security layer components
- [ ] Configure Prometheus metrics
- [ ] Set up structured logging
- [ ] Create Grafana dashboards

### Week 6 Tasks
- [ ] Performance optimization pass
- [ ] Write comprehensive documentation
- [ ] Add additional chaos tests
- [ ] Final testing and validation
- [ ] Production readiness review

This plan follows the kick-off document's timeline and ensures all components are implemented with proper testing, security, and monitoring from day one.