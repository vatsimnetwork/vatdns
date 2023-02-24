package dnshaiku

import (
	"net"
	"net/http"
	"strings"
)

func IpToIpNET(ip string) net.IP {
	return net.ParseIP(strings.Split(ip, ":")[0])
}

func GetUserIPAddressHTTP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
