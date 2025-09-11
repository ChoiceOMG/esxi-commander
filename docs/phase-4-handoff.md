# Phase 4 Handoff Document

## Executive Summary

This handoff document provides complete context for continuing Phase 4 (Advanced Features) of the ESXi Commander project. Phase 4 extends the system with enterprise features including PCI passthrough, vCenter integration, hot backups, multi-host support, advanced networking, and resource optimization.

**Current Status**: PCI passthrough foundation complete (40%), remaining features not started.

## Project Context

### What is ESXi Commander?
ESXi Commander (`ceso`) is a production-grade CLI tool for managing Ubuntu VMs on standalone ESXi hosts without vCenter. It provides simplified operations with enterprise features like monitoring, security sandboxing, and audit logging.

### Architecture Overview
```
CLI (Cobra) → Operations Layer → Govmomi/SSH → ESXi Host
                ↓
            BoltDB (state)
```

### Key Technologies
- **Language**: Go 1.21+
- **ESXi SDK**: govmomi (VMware vSphere Go SDK)
- **CLI**: Cobra + Viper
- **Database**: BoltDB (embedded)
- **Monitoring**: Prometheus metrics
- **Logging**: zerolog (structured JSON)

## Phase 4 Objectives

Transform ESXi Commander into an enterprise-grade solution supporting:
1. GPU/NIC passthrough for specialized workloads
2. vCenter integration for existing deployments
3. Hot backups without downtime
4. Multi-ESXi host management
5. Advanced networking (VLANs, DVS)
6. Intelligent resource scheduling

## Current Implementation Status

### ✅ Completed (40% of PCI Passthrough)

#### Files Created
1. **`pkg/esxi/pci/types.go`** - PCI device type definitions
   - `Device` struct with full PCI information
   - `GPUDevice` for GPU-specific features
   - `PassthroughConfig` for assignments
   - Helper methods (`IsGPU()`, `CanPassthrough()`)

2. **`pkg/esxi/pci/discovery.go`** - Device discovery implementation
   - `ListDevices()` - Enumerate all PCI devices
   - `ListGPUs()` - GPU-specific listing
   - `EnablePassthrough()` - Enable device passthrough
   - `ValidateDeviceForVM()` - Pre-assignment validation

3. **`pkg/esxi/client/client.go`** - Added `DefaultHost()` method
   - Required for PCI operations
   - Returns first available ESXi host

#### Key Design Decisions Made
- Using govmomi native APIs (no SSH fallback for PCI)
- GPU vendor detection (NVIDIA/AMD/Intel)
- Static GPU memory lookup (needs dynamic query)
- vGPU profile support preparation

### ❌ Not Started (Remaining 60% of Phase 4)

1. **PCI Passthrough Completion** (60% remaining)
2. **vCenter Integration** (0%)
3. **Hot Backup** (0%)
4. **Multi-ESXi Host** (0%)
5. **Advanced Networking** (0%)
6. **Resource Optimization** (0%)

## Task Breakdown

### Task 1: Complete PCI Passthrough (Priority 1)

#### 1.1 Create CLI Commands
Create `pkg/cli/pci/pci.go`:
```go
// Commands to implement:
ceso pci list [--gpus] [--assignable]
ceso pci info <device-id>
ceso pci enable <device-id>
ceso pci disable <device-id>
```

#### 1.2 VM Device Attachment
Create `pkg/esxi/pci/attachment.go`:
```go
func AttachDevice(ctx, vmName, deviceID string) error
func DetachDevice(ctx, vmName, deviceID string) error
func ListVMDevices(ctx, vmName string) ([]*Device, error)
```

#### 1.3 Update VM Creation
Modify `pkg/esxi/vm/operations.go`:
- Add `--gpu` flag to VM create command
- Add PCI device to VM config spec
- Validate device availability

#### 1.4 Testing
Create `pkg/esxi/pci/discovery_test.go`:
- Mock PCI device responses
- Test GPU detection logic
- Validate passthrough operations

### Task 2: vCenter Integration (Priority 2)

#### 2.1 Create vCenter Client
Create `pkg/vcenter/client.go`:
```go
type VCenterClient struct {
    client *govmomi.Client
    UseVCenter bool
}

func NewVCenterClient(config *Config) (*VCenterClient, error)
func (c *VCenterClient) IsVCenter() bool
```

#### 2.2 Resource Pool Operations
Create `pkg/vcenter/resourcepool.go`:
- List resource pools
- Create/delete resource pools
- Assign VMs to pools

#### 2.3 Configuration
Update `pkg/config/config.go`:
```yaml
vcenter:
  enabled: false
  host: "vcenter.local"
  datacenter: "DC1"
  cluster: "Cluster1"
```

### Task 3: Hot Backup with Snapshots (Priority 3)

#### 3.1 Snapshot Management
Create `pkg/backup/snapshot.go`:
```go
func CreateSnapshot(vm, name string, quiesce bool) error
func ListSnapshots(vm string) ([]*Snapshot, error)
func RevertSnapshot(vm, snapID string) error
func DeleteSnapshot(vm, snapID string) error
```

#### 3.2 Hot Backup Implementation
Modify `pkg/backup/operations.go`:
- Add `--hot` flag to backup command
- Create snapshot before backup
- Export VMDK while VM running
- Delete snapshot after export

#### 3.3 Changed Block Tracking
- Enable CBT on VMs
- Track changed blocks for incremental
- Implement incremental backup chains

### Task 4: Multi-ESXi Host Support (Priority 4)

#### 4.1 Multi-Host Configuration
Create `pkg/esxi/multihost/manager.go`:
```go
type HostManager struct {
    hosts map[string]*client.ESXiClient
    defaultHost string
}
```

#### 4.2 Host Selection
- Load balancing algorithm
- Resource-based placement
- Affinity rules

#### 4.3 Cross-Host Operations
- Cold migration between hosts
- Shared storage detection
- Network compatibility checks

### Task 5: Advanced Networking (Priority 5)

#### 5.1 VLAN Management
Create `pkg/esxi/network/vlan.go`:
- Create/delete VLANs
- Assign VMs to VLANs
- List available networks

#### 5.2 Distributed Switch
Create `pkg/esxi/network/dvs.go`:
- DVS detection and listing
- Port group management
- VM port group assignment

### Task 6: Resource Optimization (Priority 6)

#### 6.1 Placement Engine
Create `pkg/scheduler/placement.go`:
- Resource scoring algorithm
- Migration recommendations
- Automatic balancing

## Code Patterns to Follow

### Error Handling Pattern
```go
if err != nil {
    metrics.RecordOperation("failure")
    audit.LogFailure(err)
    return fmt.Errorf("operation failed: %w", err)
}
```

### Audit Logging Pattern
```go
auditCtx := audit.GetLogger().LogOperation(ctx, "operation.name", params)
defer func() {
    if err != nil {
        auditCtx.Failure(err)
    } else {
        auditCtx.Success()
    }
}()
```

### Metrics Pattern
```go
start := time.Now()
defer func() {
    metrics.RecordOperation("pci.attach", status, time.Since(start))
}()
```

## Testing Requirements

### Unit Tests
- Minimum 70% coverage for new code
- Mock govmomi responses
- Test error conditions

### Integration Tests
```bash
# Tag with +build integration
go test -tags=integration ./test/integration/pci
```

### Manual Testing Commands
```bash
# Test PCI discovery
./ceso pci list --gpus

# Test GPU attachment
./ceso vm create test-gpu --template ubuntu-22.04 --gpu 0000:81:00.0

# Test vCenter mode
VCENTER_MODE=true ./ceso vm list
```

## Dependencies to Add

```bash
# Already included
go get github.com/vmware/govmomi

# May need for advanced features
go get github.com/vmware/govmomi/vapi  # For vCenter REST APIs
go get github.com/vmware/govmomi/pbm   # For storage policies
```

## Common Gotchas

1. **PCI Addresses**: Format is `0000:XX:XX.X`, not just device ID
2. **GPU Memory Detection**: Currently hardcoded, needs dynamic query via `nvidia-smi` equivalent
3. **vGPU Profiles**: Must query host for available profiles, not assume
4. **Passthrough Enable**: Requires host reboot to take effect
5. **IOMMU Groups**: Devices in same group must be passed together
6. **vCenter API**: Different from standalone ESXi, check `IsVC()` method
7. **Snapshot Quiesce**: Requires VMware Tools installed and running
8. **CBT**: Must be enabled before first backup for incremental support

## Environment Setup

### Test Environment Requirements
- ESXi 7.x or 8.x host
- GPU for passthrough testing (optional)
- vCenter 7.x or 8.x (for vCenter features)
- Ubuntu 22.04 VM template
- Shared storage for migration tests

### Development Environment
```bash
# Clone and setup
git clone https://github.com/ChoiceOMG/esxi-commander.git
cd esxi-commander
git checkout main/phase-4

# Install dependencies
go mod tidy

# Build
go build -o ceso cmd/ceso/main.go

# Run tests
go test ./...

# Test with real ESXi
export ESXI_HOST=192.168.1.100
export ESXI_USER=root
export ESXI_PASSWORD=password
./ceso vm list
```

## Documentation to Update

1. **README.md** - Add Phase 4 features
2. **docs/gpu-passthrough.md** - Create GPU guide
3. **docs/vcenter-migration.md** - vCenter migration guide
4. **docs/multi-host-setup.md** - Multi-host configuration
5. **config.example.yaml** - Add Phase 4 config options

## Success Criteria

### Functional Requirements
- [ ] GPU passthrough works for NVIDIA Tesla GPUs
- [ ] vCenter mode detects and uses vCenter APIs
- [ ] Hot backup completes without VM downtime
- [ ] Multi-host configuration supports 3+ hosts
- [ ] VLAN assignment works correctly
- [ ] Resource optimization provides valid recommendations

### Performance Requirements
- GPU attachment: <10 seconds
- Hot backup impact: <5% performance degradation
- Multi-host operation: <2s overhead
- Placement decision: <1 second

### Quality Requirements
- Test coverage: ≥70% for new code
- All CI/CD checks pass
- Documentation complete
- Integration tests pass

## Resources and References

### Documentation
- [Govmomi PCI Passthrough](https://github.com/vmware/govmomi/tree/main/examples/host)
- [vCenter REST API](https://developer.vmware.com/apis/vsphere-automation/latest/)
- [VMware CBT Programming Guide](https://www.vmware.com/support/developer/vddk/vddk-701-programming-guide.pdf)
- [vGPU Deployment Guide](https://docs.nvidia.com/grid/latest/)

### Code References
- Phase 3 completion: `git show 0062830`
- PCI foundation: `git show 177904b`
- Project requirements: `docs/kick-off-plan.md`
- Architecture: `docs/architecture.md`

### Related Projects
- [govc](https://github.com/vmware/govmomi/tree/main/govc) - Reference CLI implementation
- [terraform-provider-vsphere](https://github.com/hashicorp/terraform-provider-vsphere) - vSphere automation

## Contact and Support

### GitHub
- Repository: https://github.com/ChoiceOMG/esxi-commander
- Issues: https://github.com/ChoiceOMG/esxi-commander/issues
- Current PR: Will be created from `main/phase-4` branch

### Key Files to Review
1. `docs/phase-4-advanced-features.md` - Detailed specification
2. `docs/implementation-plan.md` - Original project plan
3. `pkg/esxi/pci/` - Current PCI implementation
4. `pkg/esxi/client/client.go` - ESXi client base

## Next Immediate Steps

1. **Complete PCI Passthrough CLI** (2-3 hours)
   - Implement CLI commands in `pkg/cli/pci/`
   - Add to root command
   - Test with real GPU if available

2. **Add VM Attachment** (3-4 hours)
   - Create attachment.go
   - Update VM operations
   - Add integration tests

3. **Start vCenter Integration** (1 day)
   - Create vcenter package
   - Detect vCenter vs ESXi
   - Implement resource pools

4. **Document Progress** (ongoing)
   - Update phase-4-progress.md
   - Add examples to README
   - Create GPU guide

## Commit Message Template

```bash
git commit -m "Phase 4: [Component] - [What was done]

- Specific change 1
- Specific change 2
- Test coverage: X%

Next: [What comes next]

🤖 Generated with Claude Code
Co-Authored-By: Claude <noreply@anthropic.com>"
```

## Final Notes

Phase 4 transforms ESXi Commander from a basic VM management tool to an enterprise-grade orchestrator. The PCI passthrough foundation is solid, but needs CLI integration and VM attachment to be useful. vCenter integration is critical for enterprise adoption. Hot backups will differentiate from basic solutions.

Focus on completing one feature fully before moving to the next. Each feature should be independently useful and tested. The modular architecture supports incremental delivery.

Good luck! The foundation is strong, and the path forward is clear.