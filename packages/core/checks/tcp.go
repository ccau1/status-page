package checks

import (
	"context"
	"fmt"
	"net"
	"time"

	"status-page/packages/core/domain"
)

// TCPChecker checks if a TCP port is open and accepting connections.
type TCPChecker struct{}

// NewTCPChecker creates a new TCPChecker.
func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

func (t *TCPChecker) Type() string {
	return "tcp"
}

func (t *TCPChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", def.Target)
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("TCP dial failed to %s: %v", def.Target, err),
		}, err
	}
	defer conn.Close()

	return domain.CheckResult{
		Success: true,
		State:   domain.StateOperational,
		Message: fmt.Sprintf("TCP connection established to %s", def.Target),
	}, nil
}

// DNSChecker checks if a host resolves via DNS.
type DNSChecker struct {
	resolver *net.Resolver
}

// NewDNSChecker creates a new DNSChecker.
func NewDNSChecker() *DNSChecker {
	return &DNSChecker{
		resolver: net.DefaultResolver,
	}
}

func (d *DNSChecker) Type() string {
	return "dns"
}

func (d *DNSChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	start := time.Now()
	ips, err := d.resolver.LookupIPAddr(ctx, def.Target)
	if err != nil || len(ips) == 0 {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("DNS lookup failed for %s: %v", def.Target, err),
		}, err
	}

	ipStrings := make([]string, len(ips))
	for i, ip := range ips {
		ipStrings[i] = ip.String()
	}

	return domain.CheckResult{
		Success: true,
		State:   domain.StateOperational,
		Message: fmt.Sprintf("DNS resolved %s to %v in %v", def.Target, ipStrings, time.Since(start)),
	}, nil
}

func init() {
	Register(NewTCPChecker())
	Register(NewDNSChecker())
}
