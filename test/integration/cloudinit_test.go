//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/r11/esxi-commander/pkg/cloudinit"
)

func TestCloudInitBuilder(t *testing.T) {
	data := &cloudinit.CloudInitData{
		Hostname: "test-vm",
		FQDN:     "test-vm.local",
		IP:       "192.168.1.100/24",
		Gateway:  "192.168.1.1",
		DNS:      []string{"8.8.8.8", "8.8.4.4"},
		SSHKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2E... test@example.com"},
	}

	guestinfo, err := cloudinit.BuildGuestinfo(data)
	require.NoError(t, err)

	assert.Contains(t, guestinfo, "guestinfo.metadata")
	assert.Contains(t, guestinfo, "guestinfo.metadata.encoding")
	assert.Equal(t, "gzip+base64", guestinfo["guestinfo.metadata.encoding"])

	assert.Contains(t, guestinfo, "guestinfo.userdata")
	assert.Contains(t, guestinfo, "guestinfo.userdata.encoding")
	assert.Equal(t, "gzip+base64", guestinfo["guestinfo.userdata.encoding"])

	assert.Contains(t, guestinfo, "guestinfo.vendordata")
	assert.Contains(t, guestinfo, "guestinfo.vendordata.encoding")
}

func TestCloudInitWithDHCP(t *testing.T) {
	data := &cloudinit.CloudInitData{
		Hostname: "test-vm",
		FQDN:     "test-vm.local",
		SSHKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2E... test@example.com"},
	}

	guestinfo, err := cloudinit.BuildGuestinfo(data)
	require.NoError(t, err)

	assert.NotEmpty(t, guestinfo["guestinfo.metadata"])
	assert.NotEmpty(t, guestinfo["guestinfo.userdata"])
	assert.Empty(t, guestinfo["guestinfo.vendordata"])
}

func TestCloudInitWithCustomUserData(t *testing.T) {
	customUserData := `#cloud-config
packages:
  - nginx
  - postgresql-client
runcmd:
  - systemctl start nginx
`

	data := &cloudinit.CloudInitData{
		Hostname: "test-vm",
		FQDN:     "test-vm.local",
		UserData: customUserData,
	}

	guestinfo, err := cloudinit.BuildGuestinfo(data)
	require.NoError(t, err)

	assert.NotEmpty(t, guestinfo["guestinfo.userdata"])
}