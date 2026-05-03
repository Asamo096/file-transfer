package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}

func GetExternalIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return GetLocalIP()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GetLocalIP()
	}
	buf := make([]byte, 64)
	n, err := resp.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return GetLocalIP()
	}
	ip := strings.TrimSpace(string(buf[:n]))
	if net.ParseIP(ip) == nil {
		return GetLocalIP()
	}
	return ip
}
