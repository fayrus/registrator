# ZooKeeper

[Apache ZooKeeper](https://zookeeper.apache.org/) is supported as a service registry backend.

## URI scheme

```
zookeeper://host:port
```

## Example

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    zookeeper://localhost:2181
```

## Cleanup

The `zookeeper://` backend supports `-cleanup` for registrations that include the Registrator service ID in the znode payload.

Registrations created by older versions that do not include this ID are ignored during cleanup because they cannot be matched safely to a running container.

The znode path remains based on service name and address for compatibility. Cleanup uses the service ID stored in the znode payload, not the znode path.
