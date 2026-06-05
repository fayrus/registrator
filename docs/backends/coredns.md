# CoreDNS

The `coredns://` backend writes service records to etcd in [SkyDNS format](https://github.com/skynetservices/skydns) so [CoreDNS](https://coredns.io/) can resolve them via its [`etcd` plugin](https://coredns.io/plugins/etcd/).

## Requirements

- CoreDNS with the `etcd` plugin enabled
- An etcd cluster accessible to both registrator and CoreDNS

## URI format

```
coredns://etcd-host:port/skydns?zone=service.local
```

## Example

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    coredns://localhost:2379/skydns?zone=service.local
```

With this setup, a container named `web` will be resolvable at `web.service.local`.

## TLS

The `coredns://` backend supports the same TLS environment variables as `etcd://`:

| Variable | Description |
|----------|-------------|
| `ETCD_CERT_FILE` | Path to client certificate |
| `ETCD_KEY_FILE` | Path to client key |
| `ETCD_CA_CERT_FILE` | Path to CA certificate |

!!! note
    The SkyDNS2 backend was removed in v8.0.4. The `coredns://` backend is its replacement and is available since v9.0.0.
