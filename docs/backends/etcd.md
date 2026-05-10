# etcd

Two etcd backends are available depending on your cluster version and requirements.

## etcd:// — legacy

Uses the etcd v2 API. Suitable for older clusters.

```sh
fayrus/registrator:latest etcd://localhost:2379
```

## etcd2:// — recommended

Uses the official etcd v3 client (`go.etcd.io/etcd/client/v3`). Supports multiple endpoints and TLS.

```sh
fayrus/registrator:latest etcd2://localhost:2379
```

### Multiple endpoints

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  -e ETCD_ENDPOINTS=etcd2:2379,etcd3:2379 \
  fayrus/registrator:latest \
    etcd2://etcd1:2379
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
    etcd2://etcd1:2379
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `ETCD_ENDPOINTS` | Comma-separated list of additional etcd endpoints |
| `ETCD_CERT_FILE` | Path to client certificate |
| `ETCD_KEY_FILE` | Path to client key |
| `ETCD_CA_CERT_FILE` | Path to CA certificate |
