package main

import (
	_ "github.com/fayrus/registrator/backends/consul"     // registers consul:// and consul-unix:// adapters
	_ "github.com/fayrus/registrator/backends/consulkv"   // registers consulkv:// and consulkv-unix:// adapters
	_ "github.com/fayrus/registrator/backends/coredns"    // registers coredns:// adapter
	_ "github.com/fayrus/registrator/backends/etcd"        // registers etcd:// adapter (gRPC v3)
	_ "github.com/fayrus/registrator/backends/etcd-legacy" // registers etcd-legacy:// adapter (HTTP v2, legacy)
	_ "github.com/fayrus/registrator/backends/zookeeper"  // registers zookeeper:// adapter
)
