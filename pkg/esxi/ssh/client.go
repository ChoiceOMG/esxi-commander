package ssh

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	host   string
	user   string
	client *ssh.Client
}

func NewSSHClient(host, user, keyPath string) (*SSHClient, error) {
	key, err := os.ReadFile(keyPath)
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
		Timeout:         10 * time.Second,
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

func NewSSHClientWithPassword(host, user, password string) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
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

func (c *SSHClient) PowerOffVM(vmID string) error {
	_, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/power.off %s", vmID))
	return err
}

func (c *SSHClient) GetVMPowerState(vmID string) (string, error) {
	output, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/power.getstate %s", vmID))
	if err != nil {
		return "", err
	}
	if strings.Contains(output, "Powered on") {
		return "poweredOn", nil
	}
	return "poweredOff", nil
}

func (c *SSHClient) CloneVM(sourceVMID, destName, datastore string) error {
	cmd := fmt.Sprintf("vim-cmd vmsvc/clone %s %s %s", sourceVMID, destName, datastore)
	_, err := c.RunCommand(cmd)
	return err
}

func (c *SSHClient) DeleteVM(vmID string) error {
	_, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/destroy %s", vmID))
	return err
}

func (c *SSHClient) GetVMInfo(vmID string) (map[string]string, error) {
	output, err := c.RunCommand(fmt.Sprintf("vim-cmd vmsvc/get.summary %s", vmID))
	if err != nil {
		return nil, err
	}

	info := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "name =") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				info["name"] = parts[1]
			}
		}
		if strings.Contains(line, "powerState =") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				info["powerState"] = parts[1]
			}
		}
		if strings.Contains(line, "ipAddress =") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				info["ipAddress"] = parts[1]
			}
		}
	}

	return info, nil
}

func (c *SSHClient) ListVMs() ([]map[string]string, error) {
	output, err := c.RunCommand("vim-cmd vmsvc/getallvms")
	if err != nil {
		return nil, err
	}

	var vms []map[string]string
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			vm := map[string]string{
				"id":   fields[0],
				"name": fields[1],
			}
			vms = append(vms, vm)
		}
	}

	return vms, nil
}

func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}