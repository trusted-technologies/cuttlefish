package shared

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var domainRegex = regexp.MustCompile(`^(?i)[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)

// ValidateTarget checks that the target is a valid IP address or hostname.
func ValidateTarget(target string) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("target is empty")
	}
	// Direct IP address.
	if net.ParseIP(target) != nil {
		return nil
	}
	// Hostname or FQDN.
	if domainRegex.MatchString(target) {
		return nil
	}
	// Reject anything that looks like a URL or contains shell metacharacters.
	if strings.ContainsAny(target, ";&|<>$\\'\"{}[]()") {
		return fmt.Errorf("target contains invalid characters")
	}
	if _, err := url.Parse(target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return fmt.Errorf("target is not a valid IP or hostname")
}

// Clamp returns v constrained to [min, max].
func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
