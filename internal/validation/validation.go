package validation

import (
	"fmt"
	"net"
	"regexp"
)

func ValidateVMName(name string) error {
	if name == "" {
		return fmt.Errorf("VM name cannot be empty")
	}

	// VM name should be alphanumeric with hyphens and underscores
	matched, err := regexp.MatchString("^[a-zA-Z0-9-_]+$", name)
	if err != nil {
		return err
	}

	if !matched {
		return fmt.Errorf("VM name must contain only alphanumeric characters, hyphens, and underscores")
	}

	return nil
}

func ValidateIP(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address format")
	}

	return nil
}

func ValidateCIDR(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("CIDR cannot be empty")
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %v", err)
	}

	return nil
}
