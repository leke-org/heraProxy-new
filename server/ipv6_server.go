//go:build ipv6_server
// +build ipv6_server

package server

func Start() error {
	return newManager().StartIpv6()
}

func Stop() {
	newManager().Stop()
}
