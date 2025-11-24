# Caddy DNSLink Module

This Caddy module implements a DNSLink resolver that routes requests to different reverse proxies based on the content of `_dnslink` TXT records.

More information about the DNSLink specification can be found here: https://dnslink.dev

## Features

- Looks up `_dnslink.<host>` TXT records.
- Parses `dnslink=<value>`.
- Matches the value against configured prefixes.
- Rewrites the request path by prepending the DNSLink value.
- Proxies the request to the configured upstream.
- Caches DNS lookups.

## Build

To build Caddy with this module, use [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build \
    --with github.com/o8is/caddy-dnslink
```

To build with local changes:

```bash
xcaddy build \
    --with github.com/o8is/caddy-dnslink=.
```

## Docker

You can use the pre-built image `ghcr.io/o8is/caddy-dnslink`.

### Running with Docker

To run the image with your own `Caddyfile`, mount it as a volume:

```bash
docker run -d \
    -p 80:80 \
    -p 443:443 \
    -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
    ghcr.io/o8is/caddy-dnslink
```

### Custom Docker Image

To create a custom Docker image with your `Caddyfile` baked in, you can copy the Caddy binary from the image:

```dockerfile
FROM ghcr.io/o8is/caddy-dnslink AS source

FROM caddy:2.7.5

COPY --from=source /usr/bin/caddy /usr/bin/caddy
COPY Caddyfile /etc/caddy/Caddyfile
```

Then build and run your custom image:

```bash
docker build -t my-caddy-dnslink .
docker run -d -p 80:80 -p 443:443 my-caddy-dnslink
```

### Build from Source

If you want to build the image yourself:

```bash
docker build -t caddy-dnslink .
```

## Configuration

### Caddyfile

```caddyfile
{
    order dnslink before reverse_proxy
}

:80 {
    dnslink {
        proxies {
            # prefix [replacement] upstream
            /swarm /bzz varnish:8080
            /ipfs       ipfs:8080
        }
        cache_ttl 5m
    }
}
```

### JSON

```json
{
    "handler": "dnslink",
    "upstreams": {
        "/swarm": "varnish:8080",
        "/ipfs": "ipfs:8080"
    },
    "replacements": {
        "/swarm": "/bzz"
    },
    "cache_ttl": 300000000000
}
```

## How it works

1. A request comes in for `example.com`.
2. The module looks up `TXT _dnslink.example.com`.
3. Suppose it returns `dnslink=/swarm/1234...`.
4. The module checks if `/swarm` is in the `proxies` list.
5. It finds `varnish:8080`.
6. It rewrites the request path to `/bzz/1234...` (plus the original path).
7. It proxies the request to `varnish:8080`.
