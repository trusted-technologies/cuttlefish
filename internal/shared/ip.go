package shared

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// DetectIPs returns the first non-loopback IPv4 and IPv6 addresses found on local interfaces.
func DetectIPs() (ipv4, ipv6 string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", fmt.Errorf("failed to list interfaces: %w", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			if ip.To4() != nil && ipv4 == "" {
				ipv4 = ip.String()
			} else if ip.To4() == nil && ipv6 == "" {
				ipv6 = ip.String()
			}
		}
	}
	return ipv4, ipv6, nil
}

// RemoteIP extracts the client IP from a request, respecting X-Forwarded-For.
func RemoteIP(r *http.Request) string {
	fwd := r.Header.Get("X-Forwarded-For")
	if fwd != "" {
		parts := strings.Split(fwd, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if real := r.Header.Get("X-Real-Ip"); real != "" {
		return real
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

// IsIPv6 reports whether addr is an IPv6 address.
func IsIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.To4() == nil
}
