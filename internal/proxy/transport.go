package proxy

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MitmBypassHosts are hostnames intercepted via /etc/hosts during local MITM proxying.
// When making direct requests to upstream APIs, these hostnames must resolve to their
// real public IP addresses instead of 127.0.0.1 (upstream proxyFetch.js parity).
var MitmBypassHosts = []string{
	"cloudcode-pa.googleapis.com",
	"daily-cloudcode-pa.googleapis.com",
	"daily-cloudcode-pa.sandbox.googleapis.com",
	"api.individual.githubcopilot.com",
	"q.us-east-1.amazonaws.com",
	"codewhisperer.us-east-1.amazonaws.com",
	"api2.cursor.sh",
}

var (
	googleDNSResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	mitmDNSCache sync.Map // hostname -> []string
)

// ShouldBypassMitmDNS reports whether the given hostname is in MitmBypassHosts.
func ShouldBypassMitmDNS(hostname string) bool {
	h := strings.ToLower(strings.TrimSpace(hostname))
	for _, target := range MitmBypassHosts {
		if h == target || strings.HasSuffix(h, "."+target) {
			return true
		}
	}
	return false
}

// MitmBypassDialContext resolves MITM-intercepted hostnames using Google public DNS
// to bypass local /etc/hosts spoofing.
func MitmBypassDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err == nil && ShouldBypassMitmDNS(host) {
		if cached, ok := mitmDNSCache.Load(host); ok {
			if ips, ok := cached.([]string); ok && len(ips) > 0 {
				addr = net.JoinHostPort(ips[0], port)
			}
		} else {
			lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			ips, err := googleDNSResolver.LookupHost(lookupCtx, host)
			cancel()
			if err == nil && len(ips) > 0 {
				mitmDNSCache.Store(host, ips)
				addr = net.JoinHostPort(ips[0], port)
			}
		}
	}
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

// NewDirectTransport returns an http.Transport configured with MITM DNS bypass.
func NewDirectTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           MitmBypassDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// ShouldBypassNoProxy checks whether targetURL matches any pattern in noProxy.
func ShouldBypassNoProxy(targetURL, noProxy string) bool {
	noProxy = strings.TrimSpace(noProxy)
	if noProxy == "" {
		return false
	}
	if noProxy == "*" {
		return true
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		host = strings.ToLower(targetURL)
	}

	patterns := strings.Split(noProxy, ",")
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if p == "*" {
			return true
		}
		if strings.HasPrefix(p, ".") {
			if strings.HasSuffix(host, p) || host == p[1:] {
				return true
			}
		}
		if host == p || strings.HasSuffix(host, "."+p) {
			return true
		}
	}
	return false
}

// BuildEdgeRelayHeaders formats headers for Vercel, Cloudflare, and Deno edge relays.
func BuildEdgeRelayHeaders(targetURL string, existingHeaders map[string]string) map[string]string {
	headers := make(map[string]string, len(existingHeaders)+2)
	for k, v := range existingHeaders {
		headers[k] = v
	}

	parsed, err := url.Parse(targetURL)
	if err == nil {
		headers["x-relay-target"] = parsed.Scheme + "://" + parsed.Host
		path := parsed.Path
		if path == "" {
			path = "/"
		}
		if parsed.RawQuery != "" {
			path += "?" + parsed.RawQuery
		}
		headers["x-relay-path"] = path
	}
	return headers
}
