package netutil

import (
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// Default http client with timeout
// https://golang.org/pkg/net/http/#pkg-examples
// Clients and Transports are safe for concurrent use by multiple goroutines.
var defaultClient = &http.Client{
	Timeout:   time.Second * 10,
	Transport: defaultTransport,
}

// https://golang.org/src/net/http/transport.go
var defaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		DualStack: true,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          10,
	IdleConnTimeout:       10 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// Get issues a GET to the specified URL - a drop-in replacement for http.Get with timeouts.
func Get(url string) (resp *http.Response, err error) {
	return defaultClient.Get(url)
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	ip, err := LocalIP()
	if err != nil {
		return 0, err
	}
	addr, err := net.ResolveTCPAddr("tcp", ip.String()+":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// LocalIP get the host machine local IP address
func LocalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if isPrivateIP(ip) {
				return ip, nil
			}
		}
	}

	return nil, errors.New("no IP")
}

// isPrivateIP return true if IP address is from a private range
func isPrivateIP(ip net.IP) bool {
	var privateIPBlocks []*net.IPNet
	for _, cidr := range []string{
		// don't check loopback ips
		//"127.0.0.0/8",    // IPv4 loopback
		//"::1/128",        // IPv6 loopback
		//"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// ExtractPort extracts the port from a host string
func ExtractPort(host string) uint {
	re := regexp.MustCompile(`(.*:)?(\d+)`)
	matches := re.FindStringSubmatch(host)
	if len(matches) == 0 {
		return 0
	}
	portStr := matches[len(matches)-1]
	port, _ := strconv.ParseUint(portStr, 10, 64)

	return uint(port)
}
