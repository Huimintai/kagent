package sandboxbackend

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NormalizeAllowedDomainHost trims an AgentHarness allowedDomains entry into a hostname or glob
// suitable for sandbox.v1.NetworkEndpoint.host. URLs and host:port forms are accepted.
func NormalizeAllowedDomainHost(raw string) (string, bool) {
	host, _, ok := NormalizeAllowedDomainEntry(raw)
	return host, ok
}

// NormalizeAllowedDomainEntry parses an AgentHarness allowedDomains entry into a (hostname, port) pair.
// port is 0 when no explicit port is found in the entry (caller should use defaults like 443/80).
// Accepted forms: bare hostname, host:port, http(s)://host[:port][/path].
func NormalizeAllowedDomainEntry(raw string) (host string, port uint32, ok bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", 0, false
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		u, err := url.Parse(s)
		if err != nil || u.Hostname() == "" {
			return "", 0, false
		}
		var p uint32
		if portStr := u.Port(); portStr != "" {
			if n, err := strconv.ParseUint(portStr, 10, 32); err == nil {
				p = uint32(n)
			}
		}
		return u.Hostname(), p, true
	}
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
		if s == "" {
			return "", 0, false
		}
	}
	if h, portStr, err := net.SplitHostPort(s); err == nil {
		var p uint32
		if n, err := strconv.ParseUint(portStr, 10, 32); err == nil {
			p = uint32(n)
		}
		s = strings.TrimSpace(h)
		if s == "" {
			return "", 0, false
		}
		return s, p, true
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, false
	}
	return s, 0, true
}
