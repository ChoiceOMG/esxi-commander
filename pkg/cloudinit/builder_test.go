package cloudinit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildGuestinfo(t *testing.T) {
	tests := []struct {
		name    string
		data    *CloudInitData
		wantErr bool
		validate func(t *testing.T, guestinfo map[string]string)
	}{
		{
			name: "complete configuration",
			data: &CloudInitData{
				Hostname: "test-vm",
				FQDN:     "test-vm.example.com",
				IP:       "192.168.1.100/24",
				Gateway:  "192.168.1.1",
				DNS:      []string{"8.8.8.8", "8.8.4.4"},
				SSHKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2E test@example.com"},
			},
			wantErr: false,
			validate: func(t *testing.T, guestinfo map[string]string) {
				assert.Contains(t, guestinfo, "guestinfo.metadata")
				assert.Contains(t, guestinfo, "guestinfo.userdata")
				assert.Contains(t, guestinfo, "guestinfo.vendordata")
				assert.Equal(t, "gzip+base64", guestinfo["guestinfo.metadata.encoding"])
				assert.Equal(t, "gzip+base64", guestinfo["guestinfo.userdata.encoding"])
				assert.Equal(t, "gzip+base64", guestinfo["guestinfo.vendordata.encoding"])

				// Verify metadata content
				metadata := decodeGuestinfo(t, guestinfo["guestinfo.metadata"])
				assert.Contains(t, metadata, "test-vm")
				assert.Contains(t, metadata, "instance-id")
				assert.Contains(t, metadata, "local-hostname")

				// Verify userdata content
				userdata := decodeGuestinfo(t, guestinfo["guestinfo.userdata"])
				assert.Contains(t, userdata, "#cloud-config")
				assert.Contains(t, userdata, "ssh-rsa AAAAB3NzaC1yc2E")
				assert.Contains(t, userdata, "ubuntu")

				// Verify network config
				network := decodeGuestinfo(t, guestinfo["guestinfo.vendordata"])
				assert.Contains(t, network, "192.168.1.100/24")
				assert.Contains(t, network, "192.168.1.1")
				assert.Contains(t, network, "8.8.8.8")
			},
		},
		{
			name: "DHCP configuration",
			data: &CloudInitData{
				Hostname: "dhcp-vm",
				FQDN:     "dhcp-vm.example.com",
				IP:       "",
				SSHKeys:  []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 test@example.com"},
			},
			wantErr: false,
			validate: func(t *testing.T, guestinfo map[string]string) {
				// Network config should be empty for DHCP
				network := decodeGuestinfo(t, guestinfo["guestinfo.vendordata"])
				assert.Empty(t, network, "Network config should be empty for DHCP")
			},
		},
		{
			name: "custom userdata",
			data: &CloudInitData{
				Hostname: "custom-vm",
				FQDN:     "custom-vm.example.com",
				IP:       "10.0.0.100/16",
				Gateway:  "10.0.0.1",
				DNS:      []string{"10.0.0.1"},
				UserData: "#cloud-config\npackages:\n  - nginx\n  - docker",
			},
			wantErr: false,
			validate: func(t *testing.T, guestinfo map[string]string) {
				userdata := decodeGuestinfo(t, guestinfo["guestinfo.userdata"])
				assert.Contains(t, userdata, "nginx")
				assert.Contains(t, userdata, "docker")
				assert.NotContains(t, userdata, "ssh_authorized_keys") // Custom userdata replaces default
			},
		},
		{
			name: "multiple SSH keys",
			data: &CloudInitData{
				Hostname: "multi-key-vm",
				SSHKeys: []string{
					"ssh-rsa AAAAB3NzaC1yc2E user1@example.com",
					"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 user2@example.com",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, guestinfo map[string]string) {
				userdata := decodeGuestinfo(t, guestinfo["guestinfo.userdata"])
				assert.Contains(t, userdata, "user1@example.com")
				assert.Contains(t, userdata, "user2@example.com")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guestinfo, err := BuildGuestinfo(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, guestinfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, guestinfo)
				if tt.validate != nil {
					tt.validate(t, guestinfo)
				}
			}
		})
	}
}

func TestBuildMetadata(t *testing.T) {
	data := &CloudInitData{
		Hostname: "test-vm",
		FQDN:     "test-vm.example.com",
	}

	metadata, err := buildMetadata(data)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata)

	// Parse YAML to verify structure
	var meta map[string]interface{}
	err = yaml.Unmarshal(metadata, &meta)
	require.NoError(t, err)

	assert.Equal(t, "iid-test-vm", meta["instance-id"])
	assert.Equal(t, "test-vm", meta["local-hostname"])
	assert.Equal(t, "test-vm", meta["hostname"])
}

func TestBuildUserdata(t *testing.T) {
	tests := []struct {
		name     string
		data     *CloudInitData
		validate func(t *testing.T, userdata []byte)
	}{
		{
			name: "default userdata with SSH keys",
			data: &CloudInitData{
				Hostname: "test-vm",
				FQDN:     "test-vm.example.com",
				SSHKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2E test@example.com"},
			},
			validate: func(t *testing.T, userdata []byte) {
				assert.Contains(t, string(userdata), "#cloud-config")
				
				// Skip the cloud-config header
				content := strings.TrimPrefix(string(userdata), "#cloud-config\n")
				
				var data map[string]interface{}
				err := yaml.Unmarshal([]byte(content), &data)
				require.NoError(t, err)

				assert.Equal(t, "test-vm", data["hostname"])
				assert.Equal(t, "test-vm.example.com", data["fqdn"])
				assert.Equal(t, true, data["manage_etc_hosts"])

				users := data["users"].([]interface{})
				user := users[0].(map[string]interface{})
				assert.Equal(t, "ubuntu", user["name"])
				assert.Equal(t, "ALL=(ALL) NOPASSWD:ALL", user["sudo"])
				
				sshKeys := user["ssh_authorized_keys"].([]interface{})
				assert.Contains(t, sshKeys[0], "ssh-rsa AAAAB3NzaC1yc2E")
			},
		},
		{
			name: "custom userdata",
			data: &CloudInitData{
				UserData: "#cloud-config\npackages:\n  - nginx\n  - docker",
			},
			validate: func(t *testing.T, userdata []byte) {
				assert.Equal(t, "#cloud-config\npackages:\n  - nginx\n  - docker", string(userdata))
			},
		},
		{
			name: "no SSH keys",
			data: &CloudInitData{
				Hostname: "test-vm",
				FQDN:     "test-vm.example.com",
				SSHKeys:  []string{},
			},
			validate: func(t *testing.T, userdata []byte) {
				assert.Contains(t, string(userdata), "#cloud-config")
				assert.Contains(t, string(userdata), "ubuntu")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userdata, err := buildUserdata(tt.data)
			assert.NoError(t, err)
			assert.NotEmpty(t, userdata)
			if tt.validate != nil {
				tt.validate(t, userdata)
			}
		})
	}
}

func TestBuildNetwork(t *testing.T) {
	tests := []struct {
		name     string
		data     *CloudInitData
		validate func(t *testing.T, network []byte)
	}{
		{
			name: "static IP configuration",
			data: &CloudInitData{
				IP:      "192.168.1.100/24",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8", "8.8.4.4"},
			},
			validate: func(t *testing.T, network []byte) {
				var config networkConfig
				err := yaml.Unmarshal(network, &config)
				require.NoError(t, err)

				assert.Equal(t, 2, config.Version)
				assert.Contains(t, config.Ethernets, "ens192")

				eth := config.Ethernets["ens192"]
				assert.Equal(t, []string{"192.168.1.100/24"}, eth.Addresses)
				assert.Equal(t, "192.168.1.1", eth.Gateway4)
				assert.NotNil(t, eth.Nameservers)
				assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, eth.Nameservers.Addresses)
			},
		},
		{
			name: "DHCP configuration",
			data: &CloudInitData{
				IP: "",
			},
			validate: func(t *testing.T, network []byte) {
				assert.Empty(t, network, "Network config should be empty for DHCP")
			},
		},
		{
			name: "static IP without DNS",
			data: &CloudInitData{
				IP:      "10.0.0.100/16",
				Gateway: "10.0.0.1",
				DNS:     []string{},
			},
			validate: func(t *testing.T, network []byte) {
				var config networkConfig
				err := yaml.Unmarshal(network, &config)
				require.NoError(t, err)

				eth := config.Ethernets["ens192"]
				assert.Equal(t, []string{"10.0.0.100/16"}, eth.Addresses)
				assert.Equal(t, "10.0.0.1", eth.Gateway4)
				assert.Nil(t, eth.Nameservers)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network, err := buildNetwork(tt.data)
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, network)
			}
		})
	}
}

func TestEncodeGuestinfo(t *testing.T) {
	testData := "This is test data for gzip and base64 encoding"

	encoded := encodeGuestinfo([]byte(testData))
	assert.NotEmpty(t, encoded)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	assert.NoError(t, err)

	// Verify gzip decompression
	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	assert.Equal(t, testData, string(decompressed))
}

func TestGuestinfoRoundTrip(t *testing.T) {
	data := &CloudInitData{
		Hostname: "test-vm",
		FQDN:     "test-vm.example.com",
		IP:       "192.168.1.100/24",
		Gateway:  "192.168.1.1",
		DNS:      []string{"8.8.8.8"},
		SSHKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2E test@example.com"},
	}

	guestinfo, err := BuildGuestinfo(data)
	require.NoError(t, err)

	// Verify all required fields are present
	requiredFields := []string{
		"guestinfo.metadata",
		"guestinfo.metadata.encoding",
		"guestinfo.userdata",
		"guestinfo.userdata.encoding",
		"guestinfo.vendordata",
		"guestinfo.vendordata.encoding",
	}

	for _, field := range requiredFields {
		assert.Contains(t, guestinfo, field, "Missing required field: %s", field)
	}

	// Verify encoding is correct
	assert.Equal(t, "gzip+base64", guestinfo["guestinfo.metadata.encoding"])
	assert.Equal(t, "gzip+base64", guestinfo["guestinfo.userdata.encoding"])
	assert.Equal(t, "gzip+base64", guestinfo["guestinfo.vendordata.encoding"])
}

// Helper function to decode guestinfo for testing
func decodeGuestinfo(t *testing.T, encoded string) string {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	return string(decompressed)
}