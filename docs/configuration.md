# Configuration

## CLI options

```
Usage of /bin/registrator:
  /bin/registrator [options] <registry URI>

  -cleanup=false:           Remove dangling services
  -deregister="always":     Deregister exited services "always" or "on-success"
  -explicit=false:          Only register containers that have SERVICE_NAME set
  -internal=false:          Use internal ports instead of published ones
  -ip="":                   IP for ports mapped to the host
  -ip-from-container=false: Use container IP instead of host IP for service registration
  -resync=0:                Frequency with which services are resynchronized
  -useIpFromLabel="":       Use IP stored in the given container label instead of the host IP
  -retry-attempts=0:        Max retry attempts to establish a connection with the backend. Use -1 for infinite retries
  -retry-interval=2000:     Interval (in milliseconds) between retry attempts
  -tags="":                 Append tags for all registered services (supports Go template)
  -ttl=0:                   TTL for services (default is no expiry)
  -ttl-refresh=0:           Frequency with which service TTLs are refreshed
```

## Cleanup support

The `-cleanup` flag removes dangling registry entries that no longer correspond to running Docker containers. Cleanup requires backend support for listing existing registrations.

| Backend | Cleanup support | Notes |
|---------|-----------------|-------|
| `consul://` | Supported | Lists services through the Consul agent API |
| `consulkv://` | Supported | Lists keys under the configured KV path |
| `etcd://` | Supported | Lists keys under the configured etcd v3 prefix |
| `etcd-legacy://` | Supported | Lists keys recursively through the legacy etcd API |
| `coredns://` | Not currently supported | SkyDNS keys do not preserve the original Registrator service ID safely |
| `zookeeper://` | Not currently supported | Znode paths do not preserve the original Registrator service ID safely |

## Service variables

Registrator reads configuration from container environment variables prefixed with `SERVICE_`.

### Identity

| Variable | Description |
|----------|-------------|
| `SERVICE_NAME` | Override the service name |
| `SERVICE_TAGS` | Comma-separated tags. Supports Go templates (e.g. `{{.Config.Hostname}}`) |
| `SERVICE_IGNORE` | Ignore this container entirely |

### Per-port variables

| Variable | Description |
|----------|-------------|
| `SERVICE_<port>_NAME` | Name for a specific port |
| `SERVICE_<port>_<protocol>_NAME` | Name for a specific port and protocol (e.g. `SERVICE_80_tcp_NAME`) |
| `SERVICE_<N>-<M>_IGNORE` | Ignore all ports in range N–M (e.g. `SERVICE_10000-20000_IGNORE=true`) |

### Health checks

| Variable | Description |
|----------|-------------|
| `SERVICE_CHECK_HTTP` | HTTP health check path |
| `SERVICE_CHECK_HTTPS` | HTTPS health check path |
| `SERVICE_CHECK_TCP` | TCP health check |
| `SERVICE_CHECK_SCRIPT` | Script health check (space-separated args) |
| `SERVICE_CHECK_GRPC` | gRPC health check |
| `SERVICE_CHECK_TTL` | TTL-based health check |
| `SERVICE_CHECK_INTERVAL` | Health check interval (default `10s`) |
| `SERVICE_CHECK_TIMEOUT` | Health check timeout |
| `SERVICE_CHECK_TLS_SKIP_VERIFY` | Skip TLS verification for health checks |

### Consul-specific

| Variable | Description |
|----------|-------------|
| `SERVICE_ENABLE_TAG_OVERRIDE` | Allow external agents to update tags without registrator overwriting them |
