package main

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// StatsCollector collects pre-transfer stats and info, and also implements the
// io.Writer interface to track the amount/rate of data written while being
// downloaded. The latter is buffered and on the Write side of the equation,
// but should generally still be pretty close to the rate we're downloading at.
type StatsCollector struct {
	TotalBytes      uint64
	CurrentSecond   int64
	CurrentSecBytes uint64
	PerSecond       []uint64
	StartTime       int64
	EndTime         int64
	// Dns represents the DNS lookup(s) before connecting.
	Dns struct {
		StartTime int64
		EndTime   int64
		Host      string
		Addrs     []net.IPAddr
	}
	// Tls represents the TLS work, if applicable
	Tls struct {
		StartTime   int64
		EndTime     int64
		Version     uint16
		ServerName  string
		CipherSuite uint16
		// TODO: include parms from tls.ConnectionState here
	}
	// Connection is just the TCP portion of the pre-transfer work
	Connection struct {
		StartTime int64
		EndTime   int64
		Protocol  string
		Address   string
		Error     error
	}
	// Session covers the whole of the pre-transfer work (DNS, TCP, TLS)
	Session struct {
		StartTime int64
		EndTime   int64
		HostPort  string
		Local     net.Addr
		Remote    net.Addr
	}
	Request struct {
		StartTime int64
		Error     error
	}
	FirstByteTime   int64
	RequestHeaders  map[string][]string
	ResponseHeaders map[string][]string
}

func (c *StatsCollector) SetRequestHeaders(h http.Header) {
	log.Println("Request Headers:")
	c.RequestHeaders = h
	for k := range h {
		log.Printf("  %s: %s\n", k, h[k])
	}
}

func (c *StatsCollector) SetResponseHeaders(h http.Header) {
	log.Println("Response Headers:")
	c.ResponseHeaders = h
	for k := range h {
		log.Printf("  %s: %s\n", k, h[k])
	}
}

func (c *StatsCollector) Write(p []byte) (int, error) {
	n := len(p)
	c.TotalBytes += uint64(n)

	// Crude breakdown per second
	curr := time.Now().Unix()
	if curr > c.CurrentSecond {
		log.Printf("%d transferred, %d bytes/s", c.TotalBytes, c.CurrentSecBytes)
		c.PerSecond = append(c.PerSecond, c.CurrentSecBytes)
		c.CurrentSecBytes = 0
		c.CurrentSecond = curr
	}
	c.CurrentSecBytes += uint64(n)

	return n, nil
}

func (c *StatsCollector) StartDns(host string) {
	now := time.Now()
	c.Dns.StartTime = now.UnixNano()
	c.Dns.Host = host
	log.Printf("DNS Request for '%s' starting", host)
}

func (c *StatsCollector) EndDns(addrs []net.IPAddr) {
	now := time.Now()
	c.Dns.EndTime = now.UnixNano()
	c.Dns.Addrs = addrs
	log.Printf("DNS Request for '%s' returned: %s", c.Dns.Host, addrs)
}

func (c *StatsCollector) WroteRequest(e error) {
	now := time.Now()
	c.Request.StartTime = now.UnixNano()
	c.Request.Error = e
	log.Printf("HTTP Request made")
}

func (c *StatsCollector) StartConnect(network string, addr string) {
	now := time.Now()
	c.Connection.StartTime = now.UnixNano()
	c.Connection.Protocol = network
	c.Connection.Address = addr
	log.Printf("Initiating %s connection to %s", strings.ToUpper(network), addr)
}

func (c *StatsCollector) EndConnect(network string, addr string, err error) {
	now := time.Now()
	c.Connection.EndTime = now.UnixNano()
	c.Connection.Protocol = network
	c.Connection.Address = addr
	c.Connection.Error = err
	if err == nil {
		log.Printf("Connection to %s succeeded", addr)
	} else {
		log.Printf("Connection to %s failed: %s", addr, err)
	}
}

func (c *StatsCollector) StartSession(hostPort string) {
	now := time.Now()
	c.Session.StartTime = now.UnixNano()
	c.Session.HostPort = hostPort
	log.Printf("Initiating session to %s", hostPort)
}

func (c *StatsCollector) GotSession(local net.Addr, remote net.Addr) {
	now := time.Now()
	c.Session.EndTime = now.UnixNano()
	c.Session.Local = local
	c.Session.Remote = remote
	log.Printf("Initiated session to %s: %s => %s",
		c.Session.HostPort,
		local, remote)
}

func (c *StatsCollector) FirstByteReceived() {
	now := time.Now()
	c.FirstByteTime = now.UnixNano()
	c.CurrentSecond = now.Unix()

	log.Printf("Received first byte")
}

func (c *StatsCollector) StartTls() {
	now := time.Now()
	c.Tls.StartTime = now.UnixNano()
	log.Printf("Initiating TLS handshake")
}

func (c *StatsCollector) EndTls(v uint16, s uint16, n string) {
	now := time.Now()
	c.Tls.EndTime = now.UnixNano()
	log.Printf("Initiated TLS handshake")
	c.Tls.Version = v
	c.Tls.CipherSuite = s
	c.Tls.ServerName = n
}

func (c *StatsCollector) Start() {
	now := time.Now()
	c.StartTime = now.UnixNano()
}

func (c *StatsCollector) Stop() {
	now := time.Now()
	c.EndTime = now.UnixNano()
}

func (c *StatsCollector) DurationNS() int64 {
	return c.EndTime - c.StartTime
}

func (c *StatsCollector) TotalBytesTransferred() uint64 {
	return c.TotalBytes
}
