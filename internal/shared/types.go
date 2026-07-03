package shared

import "time"

// SlaveInfo describes a registered slave node.
type SlaveInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PublicURL string    `json:"public_url"`
	IPv4      string    `json:"ipv4"`
	IPv6      string    `json:"ipv6"`
	Location  string    `json:"location"`
	IperfPort string    `json:"iperf_port"`
	FileSizes []string  `json:"file_sizes,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
}

// CommandRequest is sent from master to slave to start a network tool.
type CommandRequest struct {
	Target  string `json:"target"`
	IPv6    bool   `json:"ipv6"`
	Count   int    `json:"count"`
	Timeout int    `json:"timeout"` // seconds
}

// CommandResult carries one chunk of command output.
type CommandResult struct {
	Line  string `json:"line"`
	Error string `json:"error,omitempty"`
	Done  bool   `json:"done"`
}

// MyIPInfo returns the client's detected addresses.
type MyIPInfo struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
}

// RegisterRequest is sent by a slave to register with the master.
type RegisterRequest struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	PublicURL string   `json:"public_url"`
	Token     string   `json:"token"`
	IPv4      string   `json:"ipv4"`
	IPv6      string   `json:"ipv6"`
	Location  string   `json:"location"`
	IperfPort string   `json:"iperf_port"`
	FileSizes []string `json:"file_sizes,omitempty"`
}

// HeartbeatRequest is sent periodically by a slave.
type HeartbeatRequest struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	IPv4      string `json:"ipv4"`
	IPv6      string `json:"ipv6"`
}

// TestFile describes a downloadable test file exposed by a slave.
type TestFile struct {
	Name string `json:"name"`
	Size string `json:"size"`
	URL  string `json:"url"`
}
