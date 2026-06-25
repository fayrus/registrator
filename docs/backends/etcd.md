# etcd

Two etcd backends are available depending on your cluster version and requirements.

## etcd:// — recommended

Uses the official etcd v3 gRPC client (`go.etcd.io/etcd/client/v3`). Supports multiple endpoints and TLS. Works with etcd 3.x including 3.6+.

```sh
fayrus/registrator:latest etcd://localhost:2379
```

The modern `etcd://` backend supports `-cleanup` by listing service keys under the configured prefix.

### Multiple endpoints

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  -e ETCD_ENDPOINTS=etcd2:2379,etcd3:2379 \
  fayrus/registrator:latest \
    etcd://etcd1:2379
```

### TLS

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  --volume=/etc/etcd/certs:/etc/etcd/certs \
  -e ETCD_CERT_FILE=/etc/etcd/certs/client.crt \
  -e ETCD_KEY_FILE=/etc/etcd/certs/client.key \
  -e ETCD_CA_CERT_FILE=/etc/etcd/certs/ca.crt \
  fayrus/registrator:latest \
    etcd://etcd1:2379
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `ETCD_ENDPOINTS` | Comma-separated list of additional etcd endpoints |
| `ETCD_CERT_FILE` | Path to client certificate |
| `ETCD_KEY_FILE` | Path to client key |
| `ETCD_CA_CERT_FILE` | Path to CA certificate |

## etcd-legacy:// — legacy

Uses the etcd v2 HTTP API (`coreos/go-etcd`). Requires etcd ≤ 3.5.x with `--enable-v2=true`. Not compatible with etcd 3.6+, which removed the v2 REST API.

```sh
fayrus/registrator:latest etcd-legacy://localhost:2379
```

The legacy `etcd-legacy://` backend supports `-cleanup` by recursively listing service keys under the configured prefix.

## Migrating from v9.0.x

The backend URI schemes were renamed in v9.1.0 to eliminate a long-standing naming confusion:

| v9.0.x | v9.1.0+ | etcd API |
|--------|---------|----------|
| `etcd://` | `etcd-legacy://` | HTTP v2 (legacy) |
| `etcd2://` | `etcd://` | gRPC v3 (modern) |

Update your `REGISTRY_URI` or startup command accordingly. Most users running modern etcd should replace `etcd2://` with `etcd://`.
