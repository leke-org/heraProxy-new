//go:build ipv6
// +build ipv6

package server

func Start() error {
	return newManager().StartIpv6()
}

func Stop() {
	newManager().Stop()
}
