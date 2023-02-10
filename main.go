package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	var (
		// Command line flags
		noCache   = false
		uri       = ""
		outFile   = ""
		reporters = ""
	)

	flag.BoolVar(&noCache, "noCache", false, "Request that the content not come from a cache in the middle.")
	flag.StringVar(&uri, "uri", "", "URI to request (required).")
	flag.StringVar(&outFile, "outFile", "/dev/null", "File to save downloaded data to.")
	flag.StringVar(&reporters, "reporters", "", "Comma-separated list of reporters to call. Use '-reporters list' for a list.")

	flag.Parse()

	if reporters == "list" {
		// Sort the list of keys to make it prettier to read
		reps := make([]string, 0, len(reportersList))
		for k := range reportersList {
			reps = append(reps, k)
		}

		sort.Strings(reps)

		fmt.Println("List of reporters:")
		for _, k := range reps {
			// TODO: also worth spitting out Name() or Description()?
			fmt.Printf("    %s\n", k)
		}

		os.Exit(0)
	}

	if uri == "" {
		fmt.Println("No URI specified!")
		flag.Usage()
		os.Exit(1)
	}

	// http/https for now. Things like ipfs:// will come as needed.
	if !strings.HasPrefix(strings.ToLower(uri), "http://") &&
		!strings.HasPrefix(strings.ToLower(uri), "https://") {
		fmt.Println("Currently, only http:// and https:// URIs are supported")
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("Downloading '%s'\n", uri)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		// TODO: should we do more here?
		log.Fatal(fmt.Sprintf("Request for %s failed: %s", uri, err))
	}

	// Our object for tracing/counting
	httpStats := &StatsCollector{}

	// Hook into certain HTTP tracing points
	trace := &httptrace.ClientTrace{
		DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
			httpStats.StartDns(dnsInfo.Host)
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			httpStats.EndDns(dnsInfo.Addrs)
		},
		TLSHandshakeStart: func() {
			httpStats.StartTls()
		},
		TLSHandshakeDone: func(t tls.ConnectionState, err error) {
			httpStats.EndTls(t.Version, t.CipherSuite, t.ServerName)
		},
		ConnectStart: func(net string, addr string) {
			httpStats.StartConnect(net, addr)
		},
		ConnectDone: func(net string, addr string, err error) {
			httpStats.EndConnect(net, addr, err)
		},
		GetConn: func(hostPort string) {
			httpStats.StartSession(hostPort)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			httpStats.GotSession(connInfo.Conn.LocalAddr(), connInfo.Conn.RemoteAddr())
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			httpStats.WroteRequest(w.Err)
		},
		GotFirstResponseByte: func() {
			httpStats.FirstByteReceived()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	if noCache {
		// This currently sets a few headers to prevent caching, but it
		// may be worth splitting this out into separate arguments at
		// some point for more fine-grained control in testing.
		log.Println("Requesting that content not come from cache")
		req.Header.Add("Pragma", "no-cache")
		req.Header.Add("Cache-Control", "no-cache")
		req.Header.Add("Cache-Control", "no-store")
		req.Header.Add("Cache-Control", "must-revalidate")
		req.Header.Add("Expires", "0")
	}
	httpStats.SetRequestHeaders(req.Header)
	cli := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	resp, err := cli.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	httpStats.SetResponseHeaders(resp.Header)

	log.Printf("Writing retrieved data to '%s'", outFile)
	out, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}

	httpStats.Start()
	if _, err = io.Copy(out, io.TeeReader(resp.Body, httpStats)); err != nil {
		out.Close()
		panic(err)
	}
	httpStats.Stop()
	log.Printf("Total transferred: %d in %d (%f kB/s)\n",
		httpStats.TotalBytesTransferred(), httpStats.DurationNS(),
		float64(httpStats.TotalBytesTransferred())/float64(httpStats.DurationNS())*float64(1000000000)/float64(1024))

	// Write a copy of the JSON representation of the stats to the log
	j, err := json.Marshal(httpStats)
	if err != nil {
		panic(err)
	}
	log.Println(string(j))

	out.Close()

	if reporters == "" {
		os.Exit(0)
	}

	// Now process reporters. TODO: call new() and create array, and then
	// loop through each.
	fmt.Println("")
	reqReporters := strings.Split(reporters, ",")
	for _, rep := range reqReporters {
		if r, ok := reportersList[rep]; ok {
			cr, err := r.Report(httpStats)
			if err == nil {
				fmt.Printf("%s: %s\n", rep, r.Title())
				fmt.Println(r.Description())
				fmt.Println(cr)
				fmt.Println("")
			} else {
				fmt.Printf("Reporter %s failed: %s\n", rep, err)
			}
		} else {
			log.Printf("Unknown reporter '%s'", rep)
		}
	}
}
