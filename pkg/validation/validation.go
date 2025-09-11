package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// ValidateVMName validates a VM name according to VMware conventions
func ValidateVMName(name string) error {
	if name == "" {
		return fmt.Errorf("VM name cannot be empty")
	}
	
	if len(name) > 80 {
		return fmt.Errorf("VM name cannot exceed 80 characters")
	}
	
	// VMware doesn't allow these characters in VM names
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", ";", "#"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("VM name cannot contain '%s'", char)
		}
	}
	
	// Cannot start or end with spaces or dots
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("VM name cannot start or end with spaces")
	}
	
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("VM name cannot start or end with dots")
	}
	
	return nil
}

// ValidateCIDR validates a CIDR notation IP address
func ValidateCIDR(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("IP address cannot be empty")
	}
	
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %v", err)
	}
	
	// Check if it's an IPv4 address
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 addresses are supported")
	}
	
	// Check if it's a private IP range (recommended for VMs)
	if !isPrivateIP(ip) {
		return fmt.Errorf("warning: IP address is not in a private range (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)")
	}
	
	// Check reasonable subnet mask
	ones, bits := ipNet.Mask.Size()
	if ones < 8 || ones > 30 {
		return fmt.Errorf("subnet mask /%d is outside reasonable range (/8 to /30)", ones)
	}
	
	_ = bits // suppress unused variable warning
	
	return nil
}

// ValidateGateway validates a gateway IP address
func ValidateGateway(gateway string) error {
	if gateway == "" {
		return nil // Gateway is optional
	}
	
	ip := net.ParseIP(gateway)
	if ip == nil {
		return fmt.Errorf("invalid gateway IP address")
	}
	
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 gateway addresses are supported")
	}
	
	return nil
}

// ValidateDNS validates DNS server addresses
func ValidateDNS(dnsServers []string) error {
	if len(dnsServers) == 0 {
		return nil // DNS is optional
	}
	
	if len(dnsServers) > 3 {
		return fmt.Errorf("maximum of 3 DNS servers allowed")
	}
	
	for i, dns := range dnsServers {
		ip := net.ParseIP(dns)
		if ip == nil {
			return fmt.Errorf("DNS server %d has invalid IP address: %s", i+1, dns)
		}
		
		if ip.To4() == nil {
			return fmt.Errorf("only IPv4 DNS addresses are supported: %s", dns)
		}
	}
	
	return nil
}

// ValidateSSHKey validates an SSH public key format
func ValidateSSHKey(sshKey string) error {
	if sshKey == "" {
		return nil // SSH key is optional
	}
	
	// Basic SSH public key format validation
	parts := strings.Fields(sshKey)
	if len(parts) < 2 {
		return fmt.Errorf("SSH key must have at least key type and key data")
	}
	
	keyType := parts[0]
	validTypes := []string{"ssh-rsa", "ssh-dss", "ssh-ed25519", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521"}
	
	found := false
	for _, validType := range validTypes {
		if keyType == validType {
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("unsupported SSH key type: %s", keyType)
	}
	
	return nil
}

// ValidateResourceLimits validates CPU and memory resource limits
func ValidateResourceLimits(cpu int, memoryGB int) error {
	if cpu <= 0 {
		return fmt.Errorf("CPU count must be greater than 0")
	}
	
	if cpu > 128 {
		return fmt.Errorf("CPU count cannot exceed 128")
	}
	
	if memoryGB <= 0 {
		return fmt.Errorf("memory must be greater than 0 GB")
	}
	
	if memoryGB < 1 {
		return fmt.Errorf("memory must be at least 1 GB")
	}
	
	if memoryGB > 6144 {
		return fmt.Errorf("memory cannot exceed 6144 GB")
	}
	
	return nil
}

// ValidateHostname validates a hostname
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	
	if len(hostname) > 63 {
		return fmt.Errorf("hostname cannot exceed 63 characters")
	}
	
	// Hostname pattern: alphanumeric and hyphens, cannot start/end with hyphen
	pattern := `^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	matched, err := regexp.MatchString(pattern, hostname)
	if err != nil {
		return fmt.Errorf("error validating hostname: %v", err)
	}
	
	if !matched {
		return fmt.Errorf("hostname must contain only letters, numbers, and hyphens, and cannot start or end with a hyphen")
	}
	
	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	// Private IP ranges
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // Link-local
	}
	
	for _, cidr := range private {
		_, ipNet, _ := net.ParseCIDR(cidr)
		if ipNet.Contains(ip) {
			return true
		}
	}
	
	return false
}