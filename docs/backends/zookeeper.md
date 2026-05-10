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
