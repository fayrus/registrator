package bridge

import (
	"strconv"
	"strings"

	dockerapi "github.com/fsouza/go-dockerclient"
)

func mapDefault(m map[string]string, key, default_ string) string {
	v, ok := m[key]
	if !ok || v == "" {
		return default_
	}
	return v
}

// Golang regexp module does not support /(?!\\),/ syntax for spliting by not escaped comma
// Then this function is reproducing it
func recParseEscapedComma(str string) []string {
	if len(str) == 0 {
		return []string{}
	} else if str[0] == ',' {
		return recParseEscapedComma(str[1:])
	}

	offset := 0
	for len(str[offset:]) > 0 {
		index := strings.Index(str[offset:], ",")

		if index == -1 {
			break
		} else if str[offset+index-1:offset+index] != "\\" {
			return append(recParseEscapedComma(str[offset+index+1:]), str[:offset+index])
		}

		str = str[:offset+index-1] + str[offset+index:]
		offset += index
	}

	return []string{str}
}

func combineTags(tagParts ...string) []string {
	tags := make([]string, 0)
	for _, element := range tagParts {
		tags = append(tags, recParseEscapedComma(element)...)
	}
	return tags
}

var knownProtocols = map[string]bool{"tcp": true, "udp": true}

func portInRange(portRange, port string) bool {
	parts := strings.SplitN(portRange, "-", 2)
	if len(parts) != 2 {
		return false
	}
	lo, err1 := strconv.Atoi(parts[0])
	hi, err2 := strconv.Atoi(parts[1])
	p, err3 := strconv.Atoi(port)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	return p >= lo && p <= hi
}

func serviceMetaData(config *dockerapi.Config, port, portType string) (map[string]string, map[string]bool) {
	meta := config.Env
	for k, v := range config.Labels {
		meta = append(meta, k+"="+v)
	}
	metadata := make(map[string]string)
	metadataFromPort := make(map[string]bool)
	for _, kv := range meta {
		applyMetaEntry(kv, port, portType, metadata, metadataFromPort)
	}
	return metadata, metadataFromPort
}

// applyMetaEntry parses a single SERVICE_* env/label entry and writes to metadata.
func applyMetaEntry(kv, port, portType string, metadata map[string]string, metadataFromPort map[string]bool) {
	kvp := strings.SplitN(kv, "=", 2)
	if !strings.HasPrefix(kvp[0], "SERVICE_") || len(kvp) < 2 {
		return
	}
	key := strings.ToLower(strings.TrimPrefix(kvp[0], "SERVICE_"))
	if metadataFromPort[key] {
		return
	}
	portkey := strings.SplitN(key, "_", 2)
	_, err := strconv.Atoi(portkey[0])
	isExactPort := err == nil
	isRangePort := !isExactPort && strings.Contains(portkey[0], "-") && portInRange(portkey[0], port)

	if isExactPort {
		if portkey[0] != port {
			return
		}
		applyPortKey(portkey, portType, kvp[1], metadata, metadataFromPort)
		return
	}
	if isRangePort {
		applyPortKey(portkey, portType, kvp[1], metadata, metadataFromPort)
		return
	}
	metadata[key] = kvp[1]
}

// applyPortKey writes a port-specific SERVICE_<port>_[<proto>_]<key> entry to metadata.
func applyPortKey(portkey []string, portType, value string, metadata map[string]string, metadataFromPort map[string]bool) {
	if len(portkey) < 2 {
		return
	}
	// Check for SERVICE_<port>_<protocol>_<key> format
	protokey := strings.SplitN(portkey[1], "_", 2)
	if knownProtocols[protokey[0]] && len(protokey) > 1 {
		if protokey[0] != portType {
			return
		}
		metadata[protokey[1]] = value
		metadataFromPort[protokey[1]] = true
	} else {
		metadata[portkey[1]] = value
		metadataFromPort[portkey[1]] = true
	}
}

func servicePort(container *dockerapi.Container, port dockerapi.Port, published []dockerapi.PortBinding) ServicePort {
	var hp, hip, ep, ept, eip, nm string
	if len(published) > 0 {
		hp = published[0].HostPort
		hip = published[0].HostIP
	}
	if hip == "" {
		hip = "0.0.0.0"
	}

	//for overlay networks
	//detect if container use overlay network, than set HostIP into NetworkSettings.Network[string].IPAddress
	//better to use registrator with -internal flag
	nm = container.HostConfig.NetworkMode
	if nm != "bridge" && nm != "default" && nm != "host" {
		hip = container.NetworkSettings.Networks[nm].IPAddress
	}

	exposedPort := strings.Split(string(port), "/")
	ep = exposedPort[0]
	if len(exposedPort) == 2 {
		ept = exposedPort[1]
	} else {
		ept = "tcp" // default
	}

	// Nir: support docker NetworkSettings
	eip = container.NetworkSettings.IPAddress
	if eip == "" {
		for _, network := range container.NetworkSettings.Networks {
			eip = network.IPAddress
		}
	}

	return ServicePort{
		HostPort:          hp,
		HostIP:            hip,
		ExposedPort:       ep,
		ExposedIP:         eip,
		PortType:          ept,
		ContainerID:       container.ID,
		ContainerHostname: container.Config.Hostname,
		container:         container,
	}
}
