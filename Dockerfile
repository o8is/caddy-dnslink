FROM caddy:2.7.5-builder AS builder

COPY . /caddy-dnslink
RUN xcaddy build \
    --with github.com/o8is/caddy-dnslink=/caddy-dnslink

FROM caddy:2.7.5

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
