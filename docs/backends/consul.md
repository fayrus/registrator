# Consul

[HashiCorp Consul](https://www.consul.io/) is the primary supported backend.

## URI schemes

| Scheme | Description |
|--------|-------------|
| `consul://host:port` | Standard HTTP connection |
| `consul-tls://host:port` | TLS connection |
| `consul-unix:///path/to/socket` | Unix socket |

## Example

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    consul://localhost:8500
```

## Cleanup

The `consul://` backend supports `-cleanup` by listing services through the Consul agent API.

The `consulkv://` backend also supports `-cleanup` by listing keys under the configured KV path. Registrator expects keys in the same `<path>/<service-name>/<service-id>` layout it writes during registration.

## Tag override

By default, registrator overwrites all tags on re-registration. If you use external agents (e.g. Consul itself or other tools) to manage tags, set `SERVICE_ENABLE_TAG_OVERRIDE=true` on the container to prevent registrator from overwriting them.

```sh
docker run -d \
  -e SERVICE_NAME=myapp \
  -e SERVICE_ENABLE_TAG_OVERRIDE=true \
  myapp:latest
```
