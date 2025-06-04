//go:build ipv4_server
// +build ipv4_server

package server

func Start() error {
	return newManager().StartIpv4()
}

func Stop() {
	newManager().Stop()
}
