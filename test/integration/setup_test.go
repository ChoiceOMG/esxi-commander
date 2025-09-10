package integration

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/r11/esxi-commander/pkg/cli/vm"
)

// MockESXiClient provides a mock ESXi client for integration testing
type MockESXiClient struct {
	VMs        map[string]*MockVM
	Templates  map[string]*MockTemplate
	Datastores map[string]*MockDatastore
	Networks   map[string]*MockNetwork
	Connected  bool
}

// MockVM represents a mock virtual machine
type MockVM struct {
	Name       string
	UUID       string
	PowerState string
	IP         string
	CPU        int
	Memory     int
	Disk       int
	Template   string
	CreatedAt  time.Time
	GuestInfo  map[string]string
}

// MockTemplate represents a VM template
type MockTemplate struct {
	Name             string
	Path             string
	CloudInitReady   bool
	VMwareToolsReady bool
	OS               string
}

// MockDatastore represents a datastore
type MockDatastore struct {
	Name      string
	Capacity  int64
	FreeSpace int64
	Type      string
}

// MockNetwork represents a network/portgroup
type MockNetwork struct {
	Name string
	VLAN int
	Type string
}

// SetupMockESXiClient creates and initializes a mock ESXi client
func SetupMockESXiClient() *MockESXiClient {
	client := &MockESXiClient{
		VMs:        make(map[string]*MockVM),
		Templates:  make(map[string]*MockTemplate),
		Datastores: make(map[string]*MockDatastore),
		Networks:   make(map[string]*MockNetwork),
		Connected:  true,
	}

	// Add default templates
	client.Templates["ubuntu-22.04-template"] = &MockTemplate{
		Name:             "ubuntu-22.04-template",
		Path:             "[datastore1] templates/ubuntu-22.04-template/ubuntu-22.04-template.vmx",
		CloudInitReady:   true,
		VMwareToolsReady: true,
		OS:               "ubuntu64Guest",
	}

	client.Templates["ubuntu-20.04-template"] = &MockTemplate{
		Name:             "ubuntu-20.04-template",
		Path:             "[datastore1] templates/ubuntu-20.04-template/ubuntu-20.04-template.vmx",
		CloudInitReady:   true,
		VMwareToolsReady: true,
		OS:               "ubuntu64Guest",
	}

	// Add default datastores
	client.Datastores["datastore1"] = &MockDatastore{
		Name:      "datastore1",
		Capacity:  1024 * 1024 * 1024 * 1024, // 1TB
		FreeSpace: 512 * 1024 * 1024 * 1024,  // 500GB free
		Type:      "VMFS",
	}

	// Add default networks
	client.Networks["VM Network"] = &MockNetwork{
		Name: "VM Network",
		VLAN: 0,
		Type: "standard",
	}

	client.Networks["Management"] = &MockNetwork{
		Name: "Management",
		VLAN: 100,
		Type: "standard",
	}

	return client
}

// GenerateTestVM creates a test VM with specified parameters
func GenerateTestVM(name string, template string, ip string) *MockVM {
	return &MockVM{
		Name:       name,
		UUID:       generateUUID(),
		PowerState: "poweredOff",
		IP:         ip,
		CPU:        2,
		Memory:     4,
		Disk:       40,
		Template:   template,
		CreatedAt:  time.Now(),
		GuestInfo:  make(map[string]string),
	}
}

// ValidateCloudInitData validates cloud-init metadata and userdata
func ValidateCloudInitData(metadata, userdata string) error {
	// Decode base64
	metadataBytes, err := base64.StdEncoding.DecodeString(metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	userdataBytes, err := base64.StdEncoding.DecodeString(userdata)
	if err != nil {
		return fmt.Errorf("failed to decode userdata: %v", err)
	}

	// Check metadata has required fields
	metadataStr := string(metadataBytes)
	requiredMetadataFields := []string{
		"instance-id:",
		"local-hostname:",
		"network:",
	}

	for _, field := range requiredMetadataFields {
		if !strings.Contains(metadataStr, field) {
			return fmt.Errorf("metadata missing required field: %s", field)
		}
	}

	// Check userdata has cloud-config header
	userdataStr := string(userdataBytes)
	if !strings.HasPrefix(userdataStr, "#cloud-config") {
		return fmt.Errorf("userdata missing #cloud-config header")
	}

	return nil
}

// CreateTestCloudInitData generates test cloud-init data
func CreateTestCloudInitData(hostname, ip, sshKey string) (metadata, userdata string) {
	// Generate metadata
	metadataContent := fmt.Sprintf(`instance-id: %s
local-hostname: %s
network:
  version: 2
  ethernets:
    ens192:
      addresses:
        - %s
      gateway4: %s
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4
`, hostname, hostname, ip, getGateway(ip))

	// Generate userdata
	userdataContent := fmt.Sprintf(`#cloud-config
hostname: %s
fqdn: %s.local
manage_etc_hosts: true

users:
  - name: ubuntu
    ssh_authorized_keys:
      - %s
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    shell: /bin/bash

packages:
  - open-vm-tools

runcmd:
  - systemctl enable open-vm-tools
  - systemctl start open-vm-tools
`, hostname, hostname, sshKey)

	metadata = base64.StdEncoding.EncodeToString([]byte(metadataContent))
	userdata = base64.StdEncoding.EncodeToString([]byte(userdataContent))
	
	return metadata, userdata
}

// Helper function to generate a mock UUID
func generateUUID() string {
	return fmt.Sprintf("%d-%d-%d-%d",
		time.Now().Unix(),
		time.Now().Nanosecond(),
		42,
		1337)
}

// Helper function to derive gateway from IP
func getGateway(ip string) string {
	parts := strings.Split(strings.Split(ip, "/")[0], ".")
	if len(parts) == 4 {
		parts[3] = "1"
		return strings.Join(parts, ".")
	}
	return "192.168.1.1"
}

// TestHelperFunctions tests the helper functions themselves
func TestHelperFunctions(t *testing.T) {
	t.Run("SetupMockESXiClient", func(t *testing.T) {
		client := SetupMockESXiClient()
		
		if client == nil {
			t.Fatal("Failed to create mock client")
		}
		
		if len(client.Templates) < 2 {
			t.Error("Expected at least 2 templates")
		}
		
		if len(client.Datastores) < 1 {
			t.Error("Expected at least 1 datastore")
		}
		
		if !client.Connected {
			t.Error("Expected client to be connected")
		}
	})

	t.Run("GenerateTestVM", func(t *testing.T) {
		vm := GenerateTestVM("test-vm", "ubuntu-22.04-template", "192.168.1.100/24")
		
		if vm.Name != "test-vm" {
			t.Errorf("Expected name test-vm, got %s", vm.Name)
		}
		
		if vm.Template != "ubuntu-22.04-template" {
			t.Errorf("Expected template ubuntu-22.04-template, got %s", vm.Template)
		}
		
		if vm.UUID == "" {
			t.Error("Expected UUID to be generated")
		}
	})

	t.Run("CreateTestCloudInitData", func(t *testing.T) {
		metadata, userdata := CreateTestCloudInitData(
			"test-host",
			"192.168.1.100/24",
			"ssh-rsa AAAAB3... test@example.com",
		)
		
		if metadata == "" || userdata == "" {
			t.Error("Expected cloud-init data to be generated")
		}
		
		// Validate the generated data
		if err := ValidateCloudInitData(metadata, userdata); err != nil {
			t.Errorf("Generated cloud-init data failed validation: %v", err)
		}
	})
}

// MockVMOperations provides mock VM operations for testing
type MockVMOperations struct {
	Client *MockESXiClient
}

// ListVMs returns a list of mock VMs
func (m *MockVMOperations) ListVMs() ([]*vm.VM, error) {
	var vms []*vm.VM
	
	for _, mockVM := range m.Client.VMs {
		vms = append(vms, &vm.VM{
			Name:   mockVM.Name,
			Status: mockVM.PowerState,
			IP:     mockVM.IP,
			CPU:    mockVM.CPU,
			RAM:    mockVM.Memory,
		})
	}
	
	// Add default mock VMs if none exist
	if len(vms) == 0 {
		vms = []*vm.VM{
			{Name: "ubuntu-web-01", Status: "running", IP: "192.168.1.100", CPU: 2, RAM: 4},
			{Name: "ubuntu-db-01", Status: "running", IP: "192.168.1.101", CPU: 4, RAM: 8},
			{Name: "ubuntu-test", Status: "stopped", IP: "", CPU: 1, RAM: 2},
		}
	}
	
	return vms, nil
}

// CreateVM creates a new mock VM
func (m *MockVMOperations) CreateVM(name, template, ip, sshKey string) (*MockVM, error) {
	// Check if template exists
	if _, exists := m.Client.Templates[template]; !exists {
		return nil, fmt.Errorf("template %s not found", template)
	}
	
	// Check if VM already exists
	if _, exists := m.Client.VMs[name]; exists {
		return nil, fmt.Errorf("VM %s already exists", name)
	}
	
	// Create cloud-init data
	metadata, userdata := CreateTestCloudInitData(name, ip, sshKey)
	
	// Create VM
	vm := GenerateTestVM(name, template, ip)
	vm.GuestInfo["guestinfo.metadata"] = metadata
	vm.GuestInfo["guestinfo.userdata"] = userdata
	vm.GuestInfo["guestinfo.metadata.encoding"] = "base64"
	vm.GuestInfo["guestinfo.userdata.encoding"] = "base64"
	
	// Add to client
	m.Client.VMs[name] = vm
	
	return vm, nil
}

// CloneVM clones an existing VM
func (m *MockVMOperations) CloneVM(source, dest, ip string) (*MockVM, error) {
	// Check if source exists
	sourceVM, exists := m.Client.VMs[source]
	if !exists {
		return nil, fmt.Errorf("source VM %s not found", source)
	}
	
	// Check if dest already exists
	if _, exists := m.Client.VMs[dest]; exists {
		return nil, fmt.Errorf("destination VM %s already exists", dest)
	}
	
	// Create clone
	clone := &MockVM{
		Name:       dest,
		UUID:       generateUUID(),
		PowerState: "poweredOff",
		IP:         ip,
		CPU:        sourceVM.CPU,
		Memory:     sourceVM.Memory,
		Disk:       sourceVM.Disk,
		Template:   sourceVM.Template,
		CreatedAt:  time.Now(),
		GuestInfo:  make(map[string]string),
	}
	
	// Update cloud-init for new IP
	metadata, userdata := CreateTestCloudInitData(dest, ip, "ssh-rsa AAAAB3... clone@example.com")
	clone.GuestInfo["guestinfo.metadata"] = metadata
	clone.GuestInfo["guestinfo.userdata"] = userdata
	
	// Add to client
	m.Client.VMs[dest] = clone
	
	return clone, nil
}

// DeleteVM deletes a VM
func (m *MockVMOperations) DeleteVM(name string) error {
	if _, exists := m.Client.VMs[name]; !exists {
		return fmt.Errorf("VM %s not found", name)
	}
	
	delete(m.Client.VMs, name)
	return nil
}