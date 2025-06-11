//go:build ipv4
// +build ipv4

package server

func Start() error {
	return newManager().StartIpv4()
}

func Stop() {
	newManager().Stop()
}
