package log

import (
	"net"
	"strings"
	"time"
)

type DNSLog struct {
	Server  string
	Domain  string
	Result  []net.IP
	Status  dnsStatus
	Elapsed time.Duration
	Error   error
}

func (l *DNSLog) String() string {
	var builder strings.Builder

	// Server got answer: domain -> [ip1, ip2] 23ms
	builder.WriteString(l.Server)
	builder.WriteByte(' ')
	builder.WriteString(string(l.Status))
	builder.WriteByte(' ')
	builder.WriteString(l.Domain)
	builder.WriteString(" -> [")
	builder.WriteString(joinNetIP(l.Result))
	builder.WriteByte(']')

	if l.Elapsed > 0 {
		builder.WriteByte(' ')
		builder.WriteString(l.Elapsed.String())
	}
	if l.Error != nil {
		builder.WriteString(" <")
		builder.WriteString(l.Error.Error())
		builder.WriteByte('>')
	}
	return builder.String()
}

type dnsStatus string

var (
	DNSQueried        = dnsStatus("got answer:")
	DNSCacheHit       = dnsStatus("cache HIT:")
	DNSCacheOptimiste = dnsStatus("cache OPTIMISTE:")
)

func joinNetIP(ips []net.IP) string {
	switch len(ips) {
	case 0:
		return ""
	case 1:
		return ips[0].String()
	default:
	}
	var builder strings.Builder
	builder.Grow(len(ips) * 20)
	for i := range len(ips) - 1 {
		builder.WriteString(ips[i].String())
		builder.WriteString(", ")
	}
	builder.WriteString(ips[len(ips)-1].String())
	return builder.String()
}
