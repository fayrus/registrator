package main

import (
	_ "github.com/fayrus/registrator/backends/consul"
	_ "github.com/fayrus/registrator/backends/consulkv"
	_ "github.com/fayrus/registrator/backends/etcd"
	_ "github.com/fayrus/registrator/backends/etcd2"
	_ "github.com/fayrus/registrator/backends/zookeeper"
)
