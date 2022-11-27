package util

import (
	"net"
	"os"
)

// Addresses returns addresses the server can bind to
func Addresses() []string {
	host, _ := os.Hostname()
	addresses, _ := net.LookupIP(host)

	var hosts []string

	for _, addr := range addresses {
		if ipv4 := addr.To4(); ipv4 != nil {
			hosts = append(hosts, ipv4.String())
		}
	}

	return hosts
}