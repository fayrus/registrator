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

With `-deregister=always`, services are deregistered whenever a container exits; with `on-success`, only when it exits with code 0, is stopped by a signal, or was already removed (`--rm`). In both modes, a container that is running or restarting again by the time its exit is processed (restart policy, `docker restart`) keeps its registration.

If the registry backend is unavailable when a service should be registered or deregistered, Registrator retries the failed call. Each failed call waits 30 seconds before its first retry, and the wait doubles after each failure up to 5 minutes; Docker events, `-ttl-refresh` and `-resync` ticks, and a 30-second timer only retry calls whose wait has elapsed, so bursts of events do not add backend calls.

- **Registrations** are retried until they succeed, as long as the container runs. When the container stops, its pending registrations are dropped.
- **Deregistrations** are retried until they succeed or 10 attempts have failed (about 30 minutes). A service that is registered again in the meantime (for example, the container restarted) is not deregistered by a later retry. After the last attempt the entry stays in the registry; `-cleanup` removes it on the next sync (at startup and on each `-resync` tick).

## Cleanup support

The `-cleanup` flag removes dangling registry entries that no longer correspond to running Docker containers. Cleanup requires backend support for listing existing registrations.

Cleanup only removes registrations whose service ID can be matched to Registrator's service ID format. Backends that cannot list registrations, or that store records without the original service ID, cannot be cleaned safely.

| Backend | Cleanup support | Notes |
|---------|-----------------|-------|
| `consul://` | Supported | Lists services through the Consul agent API |
| `consulkv://` | Supported | Lists keys under the configured KV path |
| `etcd://` | Supported | Lists keys under the configured etcd v3 prefix |
| `etcd-legacy://` | Supported | Lists keys recursively through the legacy etcd API |
| `coredns://` | Not currently supported | SkyDNS keys do not preserve the original Registrator service ID safely |
| `zookeeper://` | Supported | Lists znodes that include the Registrator service ID in their payload |

ZooKeeper cleanup applies to registrations created with payloads that include the service ID. Older ZooKeeper registrations without that field are ignored instead of being guessed.

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
