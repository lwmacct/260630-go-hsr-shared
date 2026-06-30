package requestctx

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type Request struct {
	IP         string
	Scheme     string
	Host       string
	UserAgent  string
	Method     string
	Path       string
	RemoteAddr string
}

type key struct{}

func ContextWithRequest(ctx context.Context, request Request) context.Context {
	return context.WithValue(ctx, key{}, request)
}

func RequestFromContext(ctx context.Context) (Request, bool) {
	request, ok := ctx.Value(key{}).(Request)
	return request, ok
}

type Middleware struct {
	trustedProxies []netip.Prefix
}

func NewMiddleware(trustedProxies []string) Middleware {
	return Middleware{trustedProxies: ParseTrustedProxies(trustedProxies)}
}

func (m Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, ok := m.Request(r)
		if ok {
			r = r.WithContext(ContextWithRequest(r.Context(), request))
		}
		next.ServeHTTP(w, r)
	})
}

func (m Middleware) Request(r *http.Request) (Request, bool) {
	ip, ok := m.ClientIP(r)
	if !ok {
		return Request{}, false
	}
	return Request{
		IP:         ip.String(),
		Scheme:     m.Scheme(r),
		Host:       r.Host,
		UserAgent:  r.UserAgent(),
		Method:     r.Method,
		Path:       r.URL.Path,
		RemoteAddr: r.RemoteAddr,
	}, true
}

func (m Middleware) Scheme(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if !m.trustedRemote(r) {
		return scheme
	}
	if proto, ok := ParseForwardedProto(r.Header.Get("X-Forwarded-Proto")); ok {
		return proto
	}
	if proto, ok := ParseForwardedProto(r.Header.Get("X-Forwarded-Scheme")); ok {
		return proto
	}
	return scheme
}

func (m Middleware) ClientIP(r *http.Request) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP, ok := ParseIP(host)
	if !ok {
		return netip.Addr{}, false
	}
	if len(m.trustedProxies) == 0 || !IPInPrefixes(remoteIP, m.trustedProxies) {
		return remoteIP, true
	}
	if ip, ok := ParseXForwardedFor(r.Header.Get("X-Forwarded-For")); ok {
		return ip, true
	}
	if ip, ok := ParseIP(r.Header.Get("X-Real-IP")); ok {
		return ip, true
	}
	return remoteIP, true
}

func (m Middleware) trustedRemote(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP, ok := ParseIP(host)
	if !ok {
		return false
	}
	return len(m.trustedProxies) > 0 && IPInPrefixes(remoteIP, m.trustedProxies)
}

func ParseTrustedProxies(values []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(value); err == nil {
			prefixes = append(prefixes, prefix)
			continue
		}
		if addr, ok := ParseIP(value); ok {
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
		}
	}
	return prefixes
}

func IPInPrefixes(ip netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func ParseXForwardedFor(value string) (netip.Addr, bool) {
	for _, part := range strings.Split(value, ",") {
		if ip, ok := ParseIP(part); ok {
			return ip, true
		}
	}
	return netip.Addr{}, false
}

func ParseForwardedProto(value string) (string, bool) {
	for _, part := range strings.Split(value, ",") {
		proto := strings.ToLower(strings.TrimSpace(strings.Trim(part, `"`)))
		switch proto {
		case "http", "https":
			return proto, true
		}
	}
	return "", false
}

func ParseIP(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(strings.Trim(value, `"`))
	if value == "" || strings.EqualFold(value, "unknown") {
		return netip.Addr{}, false
	}
	if ip, err := netip.ParseAddr(value); err == nil {
		return ip, true
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if ip, err := netip.ParseAddr(host); err == nil {
			return ip, true
		}
	}
	return netip.Addr{}, false
}
