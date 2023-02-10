package main

import (
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"strings"
)

// Maintain a map of defined reporters that may be called
var reportersList = map[string]Reporter{
	"Connection": ConnectionReporter{},
	"Header":     HeaderReporter{},
	"IPFSGW":     IpfsGwReporter{},
	"Saturn":     SaturnReporter{},
}

// An interface for code that wishes to do post-processing on the data
// gathered during the request lifetime.
type Reporter interface {
	Report(*StatsCollector) (string, error)
	Title() string
	Description() string
}

// Reporter that summarises the session init (DNS, TCP, TLS)
type ConnectionReporter struct{}

// Convenience function to take a diff an end (in ns) and a start (in ns) and
// return the difference as a float in seconds.
func (r ConnectionReporter) NsDiffInSeconds(e int64, s int64) float64 {
	return (float64(e) - float64(s)) / float64(1000000000)
}

func (r ConnectionReporter) Title() string {
	return "Session Establishment"
}

func (r ConnectionReporter) Description() string {
	return "Shows the timing for various stages of establishment of a HTTP/HTTPS session"
}

func (r ConnectionReporter) Report(s *StatsCollector) (ret string, e error) {
	tw := &strings.Builder{}
	t := tablewriter.NewWriter(tw)
	t.SetHeader([]string{"DNS Lookup", "Connection", "TLS", "Request", "First Byte"})

	data := []string{
		fmt.Sprintf("%f", r.NsDiffInSeconds(s.Dns.EndTime, s.Dns.StartTime)),
		fmt.Sprintf("%f", r.NsDiffInSeconds(s.Connection.EndTime, s.Connection.StartTime)),
		fmt.Sprintf("%f", r.NsDiffInSeconds(s.Tls.EndTime, s.Connection.StartTime)),
		fmt.Sprintf("%f", r.NsDiffInSeconds(s.Request.StartTime, s.Session.EndTime)),
		fmt.Sprintf("%f", r.NsDiffInSeconds(s.FirstByteTime, s.Request.StartTime))}
	hints := []string{
		fmt.Sprintf("%s\n%s", s.Dns.Host, s.Dns.Addrs),
		fmt.Sprintf("%s", s.Connection.Address),
		fmt.Sprintf("ver: %x\nname: %s", s.Tls.Version, s.Tls.ServerName),
		"",
		"",
	}

	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetAutoMergeCells(true)
	t.SetRowLine(true)
	t.Append(data)
	t.Append(hints)
	t.Render()

	ret = tw.String()
	return // ret, e
}

// HeaderReporter shows various request and response headers
type HeaderReporter struct{}

func (r HeaderReporter) Title() string {
	return "Request and Response Headers"
}

func (r HeaderReporter) Description() string {
	return "Shows Request and Response headers from a HTTP/HTTPS request"
}

func (r HeaderReporter) Report(s *StatsCollector) (ret string, e error) {
	tw := &strings.Builder{}
	t := tablewriter.NewWriter(tw)
	t.SetHeader([]string{"", "Key", "Value"})
	for k := range s.RequestHeaders {
		for _, v := range s.RequestHeaders[k] {
			t.Append([]string{"Request", k, v})
		}
	}
	for k := range s.ResponseHeaders {
		for _, v := range s.ResponseHeaders[k] {
			t.Append([]string{"Response", k, v})
		}
	}
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetAutoMergeCells(true)
	t.SetRowLine(true)
	t.Render()
	ret = tw.String()
	return
}

// IpfsReporter shows various aspects specific to IPFS
type IpfsGwReporter struct{}

func (r IpfsGwReporter) Title() string {
	return "IPFS Gateway Path"
}

func (r IpfsGwReporter) Description() string {
	return "Shows Information about the path through the IPFS Gateway"
}

func (r IpfsGwReporter) Report(s *StatsCollector) (ret string, e error) {
	if s.ResponseHeaders["X-Ipfs-Lb-Pop"] == nil {
		return "", errors.New("Header X-Ipfs-Lb-Pop is not present in response")
	}
	if s.ResponseHeaders["X-Ipfs-Pop"] == nil {
		return "", errors.New("Header X-Ipfs-Pop is not present in response")
	}
	tw := &strings.Builder{}
	t := tablewriter.NewWriter(tw)
	t.SetHeader([]string{"Client", "Gateway", "Load Balancer", "IPFS Node"})
	t.Append([]string{
		s.Session.Local.String(),
		s.Session.Remote.String(),
		s.ResponseHeaders["X-Ipfs-Lb-Pop"][0],
		s.ResponseHeaders["X-Ipfs-Pop"][0]})
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetAutoMergeCells(true)
	t.SetRowLine(true)
	t.Render()
	if s.ResponseHeaders["X-Proxy-Cache"] != nil {
		tw.Write([]byte(fmt.Sprintf("The request was an IPFS gateway cache %s\n",
			s.ResponseHeaders["X-Proxy-Cache"][0])))
	}
	ret = tw.String()
	return
}

// SaturnReporter shows various aspects specific to the Saturn web3 CDN
type SaturnReporter struct{}

func (r SaturnReporter) Title() string {
	return "Saturn CDN"
}

func (r SaturnReporter) Description() string {
	return "Shows information about Saturn CDN, where applicable"
}

func (r SaturnReporter) Report(s *StatsCollector) (ret string, e error) {
	if s.ResponseHeaders["Saturn-Transfer-Id"] == nil {
		return "", errors.New("Header Saturn-Transfer-Id not present in response")
	}
	if s.ResponseHeaders["Saturn-Node-Id"] == nil {
		return "", errors.New("Header Saturn-Node-Id not present in response")
	}
	if s.ResponseHeaders["Saturn-Node-Version"] == nil {
		return "", errors.New("Header Saturn-Node-Version not present in response")
	}
	if s.ResponseHeaders["Saturn-Cache-Status"] == nil {
		return "", errors.New("Header Saturn-Cache-Status not present in response")
	}

	tw := &strings.Builder{}
	t := tablewriter.NewWriter(tw)
	t.SetHeader([]string{"Client", "Transfer ID", "Saturn Node", "Saturn Node ID", "Node Version", "Cache Status"})
	t.Append([]string{
		s.Session.Local.String(),
		s.ResponseHeaders["Saturn-Transfer-Id"][0],
		s.Session.Remote.String(),
		s.ResponseHeaders["Saturn-Node-Id"][0],
		s.ResponseHeaders["Saturn-Node-Version"][0],
		s.ResponseHeaders["Saturn-Cache-Status"][0],
	})
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetAutoMergeCells(true)
	t.SetRowLine(true)
	t.Render()
	ret = tw.String()
	return
}
