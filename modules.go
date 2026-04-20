package main

import (
	_ "github.com/fayrus/registrator/consul"
	_ "github.com/fayrus/registrator/consulkv"
	_ "github.com/fayrus/registrator/etcd"
	_ "github.com/fayrus/registrator/skydns2"
	_ "github.com/fayrus/registrator/zookeeper"
)
