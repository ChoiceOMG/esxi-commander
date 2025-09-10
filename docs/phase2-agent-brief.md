# Phase 2 Implementation Brief - Core VM Operations

## Mission
Implement real ESXi integration and core VM operations for ESXi Commander. Transform the Phase 1 foundation into a working VMware orchestrator that can create, clone, and manage Ubuntu VMs on ESXi hosts.

## Prerequisites Check
Before starting Phase 2, verify Phase 1 completion:
```bash
# Verify project builds
go build ./cmd/ceso
./ceso --help
./ceso vm list --json

# Check test suite
go test ./...

# Verify structure exists
ls -la pkg/esxi/client/
ls -la pkg/cloudinit/
ls -la pkg/cli/vm/
```

## Phase 2 Objectives
By the end of Phase 2 (Week 2), deliver:
1. ✅ Working ESXi connection via govmomi
2. ✅ Cloud-init guestinfo injection for Ubuntu VMs
3. ✅ `vm create` command creating real VMs
4. ✅ `vm clone` with cold clone and re-IP
5. ✅ `vm delete` with safety checks
6. ✅ `vm info` showing real VM data
7. ✅ `vm list` connected to ESXi
8. ✅ SSH fallback for operations
9. ✅ Integration tests for VM lifecycle
10. ✅ Performance: VM create <90s

## Day-by-Day Implementation Schedule

### Day 1: ESXi Connection Layer
**Files to modify:**
- `pkg/esxi/client/client.go`
- `pkg/config/config.go`

### Day 2: Cloud-Init Builder
**Files to modify:**
- `pkg/cloudinit/builder.go`
- `pkg/cloudinit/network.go`
- `pkg/cloudinit/userdata.go`

### Day 3: VM Create Command
**Files to modify:**
- `pkg/cli/vm/create.go`
- `pkg/esxi/vm/operations.go`

### Day 4: VM Operations
**Files to modify:**
- `pkg/cli/vm/clone.go`
- `pkg/cli/vm/delete.go`
- `pkg/cli/vm/info.go`

### Day 5: Testing & Refinement
**Files to create:**
- `test/integration/vm_lifecycle_test.go`
- `test/integration/cloudinit_test.go`

## Technical Implementation Guide

### 1. ESXi Client Connection (`pkg/esxi/client/client.go`)

```go
package client

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/url"
    "time"
    
    "github.com/vmware/govmomi"
    "github.com/vmware/govmomi/find"
    "github.com/vmware/govmomi/session"
    "github.com/vmware/govmomi/vim25"
    "github.com/vmware/govmomi/vim25/soap"
)

type ESXiClient struct {
    client     *govmomi.Client
    datacenter *object.Datacenter
    finder     *find.Finder
    config     *Config
}

type Config struct {
    Host     string
    User     string
    Password string
    Insecure bool // Skip certificate verification
    Timeout  time.Duration
}

func NewClient(cfg *Config) (*ESXiClient, error) {
    ctx := context.Background()
    
    // Parse ESXi URL
    u, err := soap.ParseURL(fmt.Sprintf("https://%s/sdk", cfg.Host))
    if err != nil {
        return nil, fmt.Errorf("invalid host URL: %w", err)
    }
    
    // Set credentials
    u.User = url.UserPassword(cfg.User, cfg.Password)
    
    // Create client
    c, err := govmomi.NewClient(ctx, u, cfg.Insecure)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }
    
    // Create finder
    f := find.NewFinder(c.Client, true)
    
    // Get datacenter (default for standalone ESXi)
    dc, err := f.DefaultDatacenter(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get datacenter: %w", err)
    }
    
    f.SetDatacenter(dc)
    
    return &ESXiClient{
        client:     c,
        datacenter: dc,
        finder:     f,
        config:     cfg,
    }, nil
}

func (c *ESXiClient) ListVMs(ctx context.Context) ([]*VM, error) {
    vms, err := c.finder.VirtualMachineList(ctx, "*")
    if err != nil {
        return nil, fmt.Errorf("failed to list VMs: %w", err)
    }
    
    var result []*VM
    for _, vm := range vms {
        summary, err := vm.Summary(ctx)
        if err != nil {
            continue
        }
        
        result = append(result, &VM{
            Name:   summary.Config.Name,
            UUID:   summary.Config.Uuid,
            Status: string(summary.Runtime.PowerState),
            IP:     summary.Guest.IpAddress,
            CPU:    int(summary.Config.NumCpu),
            Memory: int(summary.Config.MemorySizeMB / 1024),
        })
    }
    
    return result, nil
}

func (c *ESXiClient) Close() error {
    if c.client != nil {
        return c.client.Logout(context.Background())
    }
    return nil
}
```

### 2. Cloud-Init Guestinfo Builder (`pkg/cloudinit/builder.go`)

```go
package cloudinit

import (
    "bytes"
    "compress/gzip"
    "encoding/base64"
    "fmt"
    "gopkg.in/yaml.v3"
)

type CloudInitData struct {
    Hostname   string
    FQDN       string
    IP         string // CIDR notation
    Gateway    string
    DNS        []string
    SSHKeys    []string
    UserData   string // Custom user-data
}

type networkConfig struct {
    Version int                    `yaml:"version"`
    Ethernets map[string]ethernet  `yaml:"ethernets"`
}

type ethernet struct {
    Addresses []string `yaml:"addresses"`
    Gateway4  string   `yaml:"gateway4,omitempty"`
    Nameservers *nameservers `yaml:"nameservers,omitempty"`
}

type nameservers struct {
    Addresses []string `yaml:"addresses"`
}

func BuildGuestinfo(data *CloudInitData) (map[string]string, error) {
    metadata, err := buildMetadata(data)
    if err != nil {
        return nil, fmt.Errorf("failed to build metadata: %w", err)
    }
    
    userdata, err := buildUserdata(data)
    if err != nil {
        return nil, fmt.Errorf("failed to build userdata: %w", err)
    }
    
    network, err := buildNetwork(data)
    if err != nil {
        return nil, fmt.Errorf("failed to build network: %w", err)
    }
    
    // Encode for VMware guestinfo
    return map[string]string{
        "guestinfo.metadata":          encodeGuestinfo(metadata),
        "guestinfo.metadata.encoding": "gzip+base64",
        "guestinfo.userdata":          encodeGuestinfo(userdata),
        "guestinfo.userdata.encoding": "gzip+base64",
        "guestinfo.vendordata":        encodeGuestinfo(network),
        "guestinfo.vendordata.encoding": "gzip+base64",
    }, nil
}

func buildMetadata(data *CloudInitData) ([]byte, error) {
    metadata := map[string]interface{}{
        "instance-id":    fmt.Sprintf("iid-%s", data.Hostname),
        "local-hostname": data.Hostname,
        "hostname":       data.Hostname,
    }
    
    return yaml.Marshal(metadata)
}

func buildUserdata(data *CloudInitData) ([]byte, error) {
    userdata := map[string]interface{}{
        "hostname": data.Hostname,
        "fqdn":     data.FQDN,
        "manage_etc_hosts": true,
        "users": []map[string]interface{}{
            {
                "name": "ubuntu",
                "sudo": "ALL=(ALL) NOPASSWD:ALL",
                "groups": "sudo",
                "shell": "/bin/bash",
                "ssh_authorized_keys": data.SSHKeys,
            },
        },
    }
    
    content, err := yaml.Marshal(userdata)
    if err != nil {
        return nil, err
    }
    
    return append([]byte("#cloud-config\n"), content...), nil
}

func buildNetwork(data *CloudInitData) ([]byte, error) {
    if data.IP == "" {
        return []byte{}, nil // Use DHCP
    }
    
    config := networkConfig{
        Version: 2,
        Ethernets: map[string]ethernet{
            "ens192": { // Default Ubuntu VMware interface
                Addresses: []string{data.IP},
                Gateway4:  data.Gateway,
            },
        },
    }
    
    if len(data.DNS) > 0 {
        config.Ethernets["ens192"].Nameservers = &nameservers{
            Addresses: data.DNS,
        }
    }
    
    return yaml.Marshal(config)
}

func encodeGuestinfo(data []byte) string {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    gz.Write(data)
    gz.Close()
    return base64.StdEncoding.EncodeToString(buf.Bytes())
}
```

### 3. VM Create Implementation (`pkg/cli/vm/create.go`)

```go
package vm

import (
    "context"
    "fmt"
    "time"
    
    "github.com/spf13/cobra"
    "github.com/r11/esxi-commander/pkg/esxi/client"
    "github.com/r11/esxi-commander/pkg/esxi/vm"
    "github.com/r11/esxi-commander/pkg/cloudinit"
)

var createCmd = &cobra.Command{
    Use:   "create <name>",
    Short: "Create a new VM from template",
    Args:  cobra.ExactArgs(1),
    RunE:  runCreate,
}

var (
    template string
    ip       string
    sshKey   string
    cpu      int
    memory   int
    disk     int
)

func init() {
    createCmd.Flags().StringVar(&template, "template", "", "Template to clone from (required)")
    createCmd.Flags().StringVar(&ip, "ip", "", "Static IP in CIDR notation (e.g., 192.168.1.100/24)")
    createCmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH public key for ubuntu user")
    createCmd.Flags().IntVar(&cpu, "cpu", 2, "Number of vCPUs")
    createCmd.Flags().IntVar(&memory, "memory", 4, "Memory in GB")
    createCmd.Flags().IntVar(&disk, "disk", 40, "Disk size in GB")
    
    createCmd.MarkFlagRequired("template")
}

func runCreate(cmd *cobra.Command, args []string) error {
    vmName := args[0]
    ctx := context.Background()
    
    // Connect to ESXi
    esxi, err := client.NewClient(getConfig())
    if err != nil {
        return fmt.Errorf("failed to connect to ESXi: %w", err)
    }
    defer esxi.Close()
    
    // Build cloud-init data
    cloudInitData := &cloudinit.CloudInitData{
        Hostname: vmName,
        FQDN:     fmt.Sprintf("%s.local", vmName),
        IP:       ip,
        SSHKeys:  []string{sshKey},
    }
    
    guestinfo, err := cloudinit.BuildGuestinfo(cloudInitData)
    if err != nil {
        return fmt.Errorf("failed to build cloud-init: %w", err)
    }
    
    // Create VM from template
    start := time.Now()
    
    vmOps := vm.NewOperations(esxi)
    newVM, err := vmOps.CreateFromTemplate(ctx, &vm.CreateOptions{
        Name:      vmName,
        Template:  template,
        CPU:       cpu,
        Memory:    memory * 1024, // Convert to MB
        Disk:      disk,
        Guestinfo: guestinfo,
    })
    
    if err != nil {
        return fmt.Errorf("failed to create VM: %w", err)
    }
    
    duration := time.Since(start)
    
    // Power on VM
    if err := vmOps.PowerOn(ctx, newVM); err != nil {
        return fmt.Errorf("failed to power on VM: %w", err)
    }
    
    fmt.Printf("✅ VM '%s' created successfully in %v\n", vmName, duration)
    fmt.Printf("   Template: %s\n", template)
    fmt.Printf("   Resources: %d vCPU, %d GB RAM, %d GB disk\n", cpu, memory, disk)
    if ip != "" {
        fmt.Printf("   IP: %s\n", ip)
    }
    
    return nil
}
```

### 4. VM Operations (`pkg/esxi/vm/operations.go`)

```go
package vm

import (
    "context"
    "fmt"
    
    "github.com/vmware/govmomi/object"
    "github.com/vmware/govmomi/vim25/types"
    "github.com/r11/esxi-commander/pkg/esxi/client"
)

type Operations struct {
    client *client.ESXiClient
}

type CreateOptions struct {
    Name      string
    Template  string
    CPU       int
    Memory    int // MB
    Disk      int // GB
    Guestinfo map[string]string
}

func NewOperations(c *client.ESXiClient) *Operations {
    return &Operations{client: c}
}

func (o *Operations) CreateFromTemplate(ctx context.Context, opts *CreateOptions) (*object.VirtualMachine, error) {
    // Find template
    template, err := o.client.FindVM(ctx, opts.Template)
    if err != nil {
        return nil, fmt.Errorf("template not found: %w", err)
    }
    
    // Get resource pool
    pool, err := o.client.DefaultResourcePool(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get resource pool: %w", err)
    }
    
    // Get datastore
    ds, err := o.client.DefaultDatastore(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get datastore: %w", err)
    }
    
    // Clone spec
    cloneSpec := types.VirtualMachineCloneSpec{
        Location: types.VirtualMachineRelocateSpec{
            Pool:      pool.Reference(),
            Datastore: ds.Reference(),
        },
        PowerOn: false,
        Template: false,
    }
    
    // Apply customizations
    if len(opts.Guestinfo) > 0 {
        var extraConfig []types.BaseOptionValue
        for key, value := range opts.Guestinfo {
            extraConfig = append(extraConfig, &types.OptionValue{
                Key:   key,
                Value: value,
            })
        }
        
        cloneSpec.Config = &types.VirtualMachineConfigSpec{
            NumCPUs:    int32(opts.CPU),
            MemoryMB:   int64(opts.Memory),
            ExtraConfig: extraConfig,
        }
    }
    
    // Get folder
    folder, err := o.client.DefaultFolder(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get folder: %w", err)
    }
    
    // Clone VM
    task, err := template.Clone(ctx, folder, opts.Name, cloneSpec)
    if err != nil {
        return nil, fmt.Errorf("failed to start clone: %w", err)
    }
    
    // Wait for completion
    info, err := task.WaitForResult(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("clone failed: %w", err)
    }
    
    // Return new VM
    return object.NewVirtualMachine(o.client.Client(), info.Result.(types.ManagedObjectReference)), nil
}

func (o *Operations) PowerOn(ctx context.Context, vm *object.VirtualMachine) error {
    task, err := vm.PowerOn(ctx)
    if err != nil {
        return fmt.Errorf("failed to start power on: %w", err)
    }
    
    _, err = task.WaitForResult(ctx, nil)
    if err != nil {
        return fmt.Errorf("power on failed: %w", err)
    }
    
    return nil
}

func (o *Operations) Delete(ctx context.Context, name string) error {
    vm, err := o.client.FindVM(ctx, name)
    if err != nil {
        return fmt.Errorf("VM not found: %w", err)
    }
    
    // Power off if running
    state, err := vm.PowerState(ctx)
    if err != nil {
        return fmt.Errorf("failed to get power state: %w", err)
    }
    
    if state == types.VirtualMachinePowerStatePoweredOn {
        task, _ := vm.PowerOff(ctx)
        task.WaitForResult(ctx, nil)
    }
    
    // Destroy VM
    task, err := vm.Destroy(ctx)
    if err != nil {
        return fmt.Errorf("failed to start destroy: %w", err)
    }
    
    _, err = task.WaitForResult(ctx, nil)
    if err != nil {
        return fmt.Errorf("destroy failed: %w", err)
    }
    
    return nil
}
```

### 5. SSH Fallback Implementation (`pkg/esxi/ssh/client.go`)

```go
package ssh

import (
    "fmt"
    "golang.org/x/crypto/ssh"
    "strings"
    "time"
)

type SSHClient struct {
    host   string
    user   string
    client *ssh.Client
}

func NewSSHClient(host, user, keyPath string) (*SSHClient, error) {
    key, err := ioutil.ReadFile(keyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read SSH key: %w", err)
    }
    
    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, fmt.Errorf("failed to parse SSH key: %w", err)
    }
    
    config := &ssh.ClientConfig{
        User: user,
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout: 10 * time.Second,
    }
    
    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    
    return &SSHClient{
        host:   host,
        user:   user,
        client: client,
    }, nil
}

func (c *SSHClient) RunCommand(cmd string) (string, error) {
    session, err := c.client.NewSession()
    if err != nil {
        return "", fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()
    
    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return string(output), fmt.Errorf("command failed: %w", err)
    }
    
    return strings.TrimSpace(string(output)), nil
}

// Fallback VM operations using vim-cmd
func (c *SSHClient) GetVMID(vmName string) (string, error) {
    output, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/getallvms | grep '%s' | awk '{print $1}'", vmName))
    if err != nil {
        return "", err
    }
    return output, nil
}

func (c *SSHClient) PowerOnVM(vmID string) error {
    _, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/power.on %s", vmID))
    return err
}

func (c *SSHClient) Close() error {
    if c.client != nil {
        return c.client.Close()
    }
    return nil
}
```

## Configuration Requirements

### ESXi Connection Config (`~/.ceso/config.yaml`)
```yaml
esxi:
  host: 192.168.1.10
  user: root
  password: ${ESXI_PASSWORD}  # Use environment variable
  insecure: true              # For self-signed certificates

defaults:
  template: ubuntu-22.04-template
  datastore: datastore1
  network: VM Network
  cpu: 2
  memory: 4
  disk: 40

security:
  mode: standard              # For human operators
  # mode: restricted         # For AI agents
```

### Ubuntu Template Requirements
The template must have:
1. Ubuntu 22.04 LTS or 24.04 LTS
2. cloud-init installed and configured
3. VMware Tools (open-vm-tools) installed
4. Datasource configured for VMware guestinfo:
```yaml
# /etc/cloud/cloud.cfg.d/99_guestinfo.cfg
datasource:
  VMware:
    allow_raw_data: true
datasource_list: [ VMware, None ]
```

## Testing Strategy

### Integration Test Example (`test/integration/vm_lifecycle_test.go`)
```go
func TestVMLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Create VM
    vmName := fmt.Sprintf("test-vm-%d", time.Now().Unix())
    
    cmd := exec.Command("./ceso", "vm", "create", vmName,
        "--template", "ubuntu-22.04-template",
        "--ip", "192.168.1.200/24",
        "--ssh-key", testSSHKey)
    
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "VM creation failed: %s", output)
    
    // Verify VM exists
    cmd = exec.Command("./ceso", "vm", "list", "--json")
    output, err = cmd.Output()
    require.NoError(t, err)
    
    var vms []VM
    err = json.Unmarshal(output, &vms)
    require.NoError(t, err)
    
    found := false
    for _, vm := range vms {
        if vm.Name == vmName {
            found = true
            assert.Equal(t, "running", vm.Status)
            break
        }
    }
    require.True(t, found, "VM not found in list")
    
    // Cleanup
    defer func() {
        cmd = exec.Command("./ceso", "vm", "delete", vmName)
        cmd.Run()
    }()
    
    // Test SSH connectivity
    time.Sleep(30 * time.Second) // Wait for boot
    
    // Verify cloud-init worked
    client := setupSSHClient(t, "192.168.1.200")
    hostname, err := client.RunCommand("hostname")
    require.NoError(t, err)
    assert.Equal(t, vmName, hostname)
}
```

## Performance Requirements

### VM Creation Benchmark
```go
func BenchmarkVMCreate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        // Create VM
        vmName := fmt.Sprintf("bench-vm-%d", i)
        createVM(vmName)
        
        duration := time.Since(start)
        
        // Must be under 90 seconds
        if duration > 90*time.Second {
            b.Fatalf("VM creation took %v, exceeding 90s limit", duration)
        }
        
        // Cleanup
        deleteVM(vmName)
    }
}
```

## Common Issues & Solutions

### Issue 1: Certificate Verification Failed
**Solution**: Set `insecure: true` in config for self-signed certificates

### Issue 2: Cloud-init Not Working
**Debug Steps**:
```bash
# Check guestinfo on VM
vmware-rpctool "info-get guestinfo.metadata"
vmware-rpctool "info-get guestinfo.userdata"

# Check cloud-init logs
sudo cat /var/log/cloud-init.log
sudo cloud-init status --long
```

### Issue 3: VM Creation Timeout
**Possible Causes**:
- Slow datastore (use SSD)
- Large template (use thin provisioning)
- Network issues

### Issue 4: govmomi Connection Issues
**Fallback to SSH**:
```go
if govmomiErr != nil {
    log.Warn("govmomi failed, using SSH fallback")
    return sshClient.PowerOnVM(vmID)
}
```

## Success Criteria Checklist

### Functional Requirements
- [ ] `ceso vm create` creates VM in <90s
- [ ] `ceso vm clone` performs cold clone
- [ ] `ceso vm delete` safely removes VM
- [ ] `ceso vm info` shows accurate data
- [ ] `ceso vm list` returns all VMs
- [ ] Cloud-init sets hostname correctly
- [ ] Cloud-init configures static IP
- [ ] SSH key injection works

### Non-Functional Requirements
- [ ] All operations have --dry-run support
- [ ] Errors are handled gracefully
- [ ] Operations are logged with zerolog
- [ ] SSH fallback works when API fails
- [ ] Test coverage >70%
- [ ] Integration tests pass

### Performance Metrics
- [ ] VM create: <90 seconds
- [ ] VM clone (80GB): <5 minutes
- [ ] VM list: <2 seconds
- [ ] VM delete: <30 seconds

## Deliverables for Phase 2

1. **Working Commands**:
   - `ceso vm create` with cloud-init
   - `ceso vm clone` with re-IP
   - `ceso vm delete` with safety
   - `ceso vm info` with details
   - `ceso vm list` from ESXi

2. **Core Components**:
   - ESXi client with govmomi
   - Cloud-init builder
   - SSH fallback mechanism
   - Error handling throughout

3. **Tests**:
   - Integration test suite
   - VM lifecycle tests
   - Cloud-init verification
   - Performance benchmarks

4. **Documentation**:
   - Updated README with examples
   - Configuration guide
   - Troubleshooting guide

## Next Steps (Phase 3 Preview)

After Phase 2 completion:
- Implement hot backup with snapshots
- Add VM reconfiguration (CPU/RAM/disk)
- Create backup/restore operations
- Add chaos testing scenarios
- Implement Prometheus metrics

## Support & Resources

### govmomi Examples
- [Official Examples](https://github.com/vmware/govmomi/tree/main/examples)
- [govc CLI source](https://github.com/vmware/govmomi/tree/main/govc)

### Cloud-Init Documentation
- [Ubuntu Cloud-Init](https://cloudinit.readthedocs.io/)
- [VMware Datasource](https://cloudinit.readthedocs.io/en/latest/topics/datasources/vmware.html)

### Testing Resources
- Run tests: `go test -tags=integration ./test/integration/...`
- Coverage: `go test -cover ./...`
- Benchmarks: `go test -bench=. ./...`

## Important Reminders

1. **DO NOT** skip error handling - production readiness is critical
2. **DO NOT** hardcode credentials - use config file or env vars
3. **ALWAYS** test with real ESXi before marking complete
4. **ALWAYS** implement dry-run mode for safety
5. **FOCUS** on Ubuntu 22.04 LTS first, then 24.04 LTS
6. **MAINTAIN** backward compatibility with Phase 1 structure

This brief provides everything needed to implement Phase 2. Focus on one day's tasks at a time, test thoroughly, and ensure each component works before moving to the next.