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
	Version   int                   `yaml:"version"`
	Ethernets map[string]ethernet   `yaml:"ethernets"`
}

type ethernet struct {
	Addresses   []string     `yaml:"addresses"`
	Gateway4    string       `yaml:"gateway4,omitempty"`
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
		"hostname":         data.Hostname,
		"fqdn":             data.FQDN,
		"manage_etc_hosts": true,
		"users": []map[string]interface{}{
			{
				"name":                "ubuntu",
				"sudo":                "ALL=(ALL) NOPASSWD:ALL",
				"groups":              "sudo",
				"shell":               "/bin/bash",
				"ssh_authorized_keys": data.SSHKeys,
			},
		},
	}

	if data.UserData != "" {
		return []byte(data.UserData), nil
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

	eth := ethernet{
		Addresses: []string{data.IP},
		Gateway4:  data.Gateway,
	}
	
	if len(data.DNS) > 0 {
		eth.Nameservers = &nameservers{
			Addresses: data.DNS,
		}
	}
	
	config := networkConfig{
		Version: 2,
		Ethernets: map[string]ethernet{
			"ens192": eth,
		},
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
