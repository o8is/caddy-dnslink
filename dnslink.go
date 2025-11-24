package dnslink

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	dnslinkpkg "github.com/dnslink-std/go"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(new(DNSLink))
	httpcaddyfile.RegisterHandlerDirective("dnslink", parseCaddyfile)
}

type DNSLink struct {
	// Upstreams maps a prefix (e.g. "/swarm") to a reverse proxy upstream (e.g. "varnish:8080").
	Upstreams map[string]string `json:"upstreams,omitempty"`

	// Replacements maps a prefix (e.g. "/swarm") to a replacement string (e.g. "/bzz").
	Replacements map[string]string `json:"replacements,omitempty"`

	// CacheTTL is the duration to cache DNS lookups. Default is 1 minute.
	CacheTTL caddy.Duration `json:"cache_ttl,omitempty"`

	// proxies holds the initialized reverse proxy handlers.
	proxies map[string]*reverseproxy.Handler

	// cache holds the DNS lookup results.
	cache sync.Map

	logger *zap.Logger
}

type cachedLookup struct {
	namespace  string
	identifier string
	expiresAt  time.Time
}

func (d *DNSLink) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.dnslink",
		New: func() caddy.Module { return new(DNSLink) },
	}
}

func (d *DNSLink) Provision(ctx caddy.Context) error {
	d.logger = ctx.Logger(d)
	d.proxies = make(map[string]*reverseproxy.Handler)

	if d.CacheTTL == 0 {
		d.CacheTTL = caddy.Duration(1 * time.Minute)
	}

	for prefix, upstream := range d.Upstreams {
		// Create a reverse proxy handler for this upstream
		rp := &reverseproxy.Handler{
			Upstreams: reverseproxy.UpstreamPool{
				{Dial: upstream},
			},
		}
		// We need to provision the reverse proxy
		if err := rp.Provision(ctx); err != nil {
			return fmt.Errorf("provisioning reverse proxy for %s: %v", prefix, err)
		}
		d.proxies[prefix] = rp
	}
	return nil
}

func (d *DNSLink) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	d.logger.Debug("handling request", zap.String("uri", r.RequestURI), zap.String("host", r.Host))
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	namespace, identifier, err := d.resolve(host)
	if err != nil {
		d.logger.Debug("dns lookup failed", zap.String("host", host), zap.Error(err))
		return next.ServeHTTP(w, r)
	}

	if namespace == "" {
		return next.ServeHTTP(w, r)
	}

	// Match prefix
	// We assume the prefix in Caddyfile matches /namespace
	prefix := "/" + namespace
	if proxy, ok := d.proxies[prefix]; ok {
		// Match found!
		d.logger.Debug("dnslink match", zap.String("host", host), zap.String("namespace", namespace), zap.String("identifier", identifier))

		// Construct new path
		// Start with replacement or prefix
		base := prefix
		if replacement, ok := d.Replacements[prefix]; ok {
			base = replacement
		}

		// Ensure base ends with /
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}

		// Add identifier
		newPath := base + identifier

		// Ensure identifier part ends with / if it's a directory-like structure
		if !strings.HasSuffix(newPath, "/") {
			newPath += "/"
		}

		// Append original path (stripped of leading /)
		originalPath := r.URL.Path
		cleanOriginal := strings.TrimPrefix(originalPath, "/")
		newPath += cleanOriginal

		r.URL.Path = newPath

		// Delegate to the reverse proxy
		return proxy.ServeHTTP(w, r, next)
	}

	d.logger.Debug("no matching prefix found", zap.String("host", host), zap.String("namespace", namespace))
	return next.ServeHTTP(w, r)
}

func (d *DNSLink) resolve(host string) (string, string, error) {
	if val, ok := d.cache.Load(host); ok {
		entry := val.(cachedLookup)
		if time.Now().Before(entry.expiresAt) {
			return entry.namespace, entry.identifier, nil
		}
		d.cache.Delete(host)
	}

	// Use the official dnslink library to resolve
	result, err := dnslinkpkg.Resolve(host)
	if err != nil {
		// If it's just that no link was found, we return empty string without error
		// so the handler can continue to the next middleware.
		d.logger.Debug("dnslink resolution result", zap.String("host", host), zap.Error(err))
		return "", "", nil
	}

	// Find a link
	var namespace, identifier string
	for ns, entries := range result.Links {
		if len(entries) > 0 {
			namespace = ns
			identifier = entries[0].Identifier
			break
		}
	}

	// Cache the result
	d.cache.Store(host, cachedLookup{
		namespace:  namespace,
		identifier: identifier,
		expiresAt:  time.Now().Add(time.Duration(d.CacheTTL)),
	})

	return namespace, identifier, nil
}

// parseCaddyfile parses the dnslink directive.
// Syntax:
//
//	dnslink {
//	    proxies {
//	        /swarm varnish:8080
//	        /ipfs  ipfs:8080
//	    }
//	    cache_ttl 1m
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	d := new(DNSLink)
	d.Upstreams = make(map[string]string)
	d.Replacements = make(map[string]string)

	for h.Next() {
		for h.NextBlock(0) {
			switch h.Val() {
			case "proxies":
				for h.NextBlock(1) {
					prefix := h.Val()
					if !h.NextArg() {
						return nil, h.ArgErr()
					}
					arg2 := h.Val()

					var upstream, replacement string

					if h.NextArg() {
						// 3 arguments: prefix replacement upstream
						replacement = arg2
						upstream = h.Val()
					} else {
						// 2 arguments: prefix upstream
						upstream = arg2
					}

					upstream = strings.TrimPrefix(upstream, "http://")
					upstream = strings.TrimPrefix(upstream, "https://")
					d.Upstreams[prefix] = upstream

					if replacement != "" {
						d.Replacements[prefix] = replacement
					}
				}
			case "cache_ttl":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				dur, err := caddy.ParseDuration(h.Val())
				if err != nil {
					return nil, err
				}
				d.CacheTTL = caddy.Duration(dur)
			default:
				return nil, h.Errf("unknown subdirective '%s'", h.Val())
			}
		}
	}
	return d, nil
}

// Interface guards
var (
	_ caddy.Module                = (*DNSLink)(nil)
	_ caddy.Provisioner           = (*DNSLink)(nil)
	_ caddyhttp.MiddlewareHandler = (*DNSLink)(nil)
)
