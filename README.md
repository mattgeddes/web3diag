# web3diag

## Overview

`web3diag` is a simple binary written in Go that might be helpful in diagnosing issues in the Web3 space. It primarily deals with HTTP and HTTPS today, but will likely be extended to include IPFS and others in the future. In addition to HTTP/HTTPS tracing and timing, it also has some in-built knowledge of the headers used by (The IPFS Gateway)[https://ipfs.io] and of the (Saturn Web3 CDN network)[https://strn.pl].

It should run just fine on any platform supported by Go and is frequently run on arm64, amd64 and arm and on Linux, FreeBSD and Darwin (MacOS).

## Usage

```
$ ./web3diag -help
Usage of ./web3diag:
  -noCache
    	Request that the content not come from a cache in the middle.
  -outFile string
    	File to save downloaded data to. (default "/dev/null")
  -reporters string
    	Comma-separated list of reporters to call. Use '-reporters list' for a list.
  -uri string
    	URI to request (required).
```

The `web3diag` client will retrieve the URL provided with the `-uri` flag and give a log of diagnostic output to stdout. The data itself will be discarded (written to `/dev/null` unless the `-outFile` flag is used to write it to another file.

The `-noCache` option adds headers to hint to any hops along the way that the content should not be cached or retrieved from cache. The specific headers used are:

```
  Expires: 0
  Pragma: no-cache
  Cache-Control: no-cache no-store must-revalidate
```

These may or may not be honoured by hosts along the way.

The `-reporters` flag is covered in more detail below, but allows the user to specify a builtin module for post-processing trace data. The `-reporters list` flag may be used to enumerate valid options:

```
$ ./web3diag -reporters list
List of reporters:
    Connection
    Header
    IPFSGW
    Saturn
```

## Diagnostic Output



## JSON Data

At the end of the log output, and just before executing any reporters, the trace and diagnostic data is written as a JSON object. This may be useful in processing the data offline, or comparing multiple similar runs.

## Reporters

Reporters are small pieces of functionality built into `web3diag` to do some post-processing on the request and trace data collected. Multple may be specified as a comma separated list. For example: `./web3diag -uri https://ipfs.io/ipfs/ -reporters Connection,IPFSGW`

### Connection

This reporter simply summarises where the time was spent in establishing a HTTP/HTTPS session, by breaking down DNS requests, TCP connection establishment and TLS handshaking.

```
Connection: Session Establishment
Shows the timing for various stages of establishment of a HTTP/HTTPS session
+-----------------------+----------------+---------------+----------+------------+
|      DNS LOOKUP       |   CONNECTION   |      TLS      | REQUEST  | FIRST BYTE |
+-----------------------+----------------+---------------+----------+------------+
| 0.001112              | 0.000287       | 0.908020      | 0.000126 | 0.258721   |
+-----------------------+----------------+---------------+----------+------------+
| localhost [{127.0.0.1 | 127.0.0.1:3128 | ver: 304      |          |            |
| } {::1 }]             |                | name: strn.pl |          |            |
+-----------------------+----------------+---------------+----------+------------+
```

In addition to the timing (in seconds), it also includes some basic information about the DNS request made, the TCP connection and the TLS handshake.

In the above example, the session is being proxied through a SOCKS5 proxy, which is described below.

### IPFSGW

The IPFSGW reporter summarises information specific to the public IPFS/HTTP gateway.

```
IPFSGW: IPFS Gateway Path
Shows Information about the path through the IPFS Gateway
+-------------------+-----------------+-------------------+----------------+
|      CLIENT       |     GATEWAY     |   LOAD BALANCER   |   IPFS NODE    |
+-------------------+-----------------+-------------------+----------------+
| 192.168.0.6:56985 | 209.94.90.1:443 | gateway-bank1-sg1 | ipfs-bank5-sg1 |
+-------------------+-----------------+-------------------+----------------+
The request was an IPFS gateway cache HIT

```

It includes the server endpoint address, load balancer name and backend IPFS node, and whether the request was a cache hit or miss.


### Saturn

This reporter summarises information specific to the Saturn CDN gleaned from the HTTP headers in the response.

```
Saturn: Saturn CDN
Shows information about Saturn CDN, where applicable
+-------------------+----------------------------------+-------------------+--------------------------------------+--------------+--------------+
|      CLIENT       |           TRANSFER ID            |    SATURN NODE    |            SATURN NODE ID            | NODE VERSION | CACHE STATUS |
+-------------------+----------------------------------+-------------------+--------------------------------------+--------------+--------------+
| 172.16.1.53:54757 | e3add4557d1ef10a563a4d96ee3d851d | 103.93.130.94:443 | d54286f3-7da5-42ae-8762-04d705e06354 | 531_e5eb4dc  | HIT          |
+-------------------+----------------------------------+-------------------+--------------------------------------+--------------+--------------+

```

Here we can see the Saturn node ID and endpoint address, as well as whether the request was a cache hit or cache miss.

### Headers

The Headers reporter simply shows a tabular summary of request and response headers.

```
Header: Request and Response Headers
Shows Request and Response headers from a HTTP/HTTPS request
+----------+------------------+-------------------------------+
|          |       KEY        |             VALUE             |
+----------+------------------+-------------------------------+
| Request  | Pragma           | no-cache                      |
+          +------------------+                               +
|          | Cache-Control    |                               |
+          +                  +-------------------------------+
|          |                  | no-store                      |
+          +                  +-------------------------------+
|          |                  | must-revalidate               |
+          +------------------+-------------------------------+
|          | Expires          | 0                             |
+----------+------------------+-------------------------------+
| Response | Etag             | "2d-5eed6c325ce00"            |
+          +------------------+-------------------------------+
|          | Content-Type     | text/html                     |
+          +------------------+-------------------------------+
|          | Date             | Wed, 18 Jan 2023 00:27:40 GMT |
+          +------------------+-------------------------------+
|          | Server           | Apache/2.4.54 (Unix)          |
+          +------------------+-------------------------------+
|          | Content-Location | index.html.en                 |
+          +------------------+-------------------------------+
|          | Accept-Ranges    | bytes                         |
+          +------------------+-------------------------------+
|          | Content-Length   | 45                            |
+          +------------------+-------------------------------+
|          | Vary             | negotiate                     |
+          +------------------+-------------------------------+
|          | Tcn              | choice                        |
+          +------------------+-------------------------------+
|          | Last-Modified    | Fri, 02 Dec 2022 11:37:28 GMT |
+----------+------------------+-------------------------------+
```

## Proxy Support

`web3diag` uses the `http.ProxyFromEnvironment` proxy configuration, which allows the user to specify a HTTP, HTTPS or SOCKS5 proxy server to make requests via. For example, to proxy a request via an OpenSSH SOCKS5 tunnel to a remote host, one could:

 * Establish the `ssh` connection with the `-D` option, specifying a TCP port to listen for SOCKS5 requests on:
 
 ```
myhost$ ssh -D3128 user@some.remote.host
 ```

 * Use the `https_proxy` environment variable to have `web3diag` on our local machine proxy the request through the `ssh` SOCKS5 proxy:

```
myhost$ https_proxy=socks5://localhost:3128 ./web3diag https://strn.pl/ipfs/QmaJ6kN9fW3TKpVkpf1NuW7cjhHjNp5Jwr3cQuHzsoZWkJ
```

In the above case, `web3diag` will make a request for the `strn.pl` HTTPS url provided, but via the SOCKS5 proxy provided by the `ssh -D` command. The request is then tunnelled over the `ssh` session and made from the destination (`some.remote.host`).

This can be useful for checking things from a remote region or bypassing middlemne, for example.

