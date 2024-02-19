package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/data"
	"github.com/agux/roprox/types"
	"golang.org/x/net/proxy"
)

// ConnResponseWriter is our custom ResponseWriter that uses net.Conn.
type ConnResponseWriter struct {
	conn        net.Conn
	header      http.Header
	statusCode  int
	wroteHeader bool
}

// Make sure ConnResponseWriter implements http.ResponseWriter.
var _ http.ResponseWriter = &ConnResponseWriter{}

// NewConnResponseWriter creates a new instance of ConnResponseWriter.
func NewConnResponseWriter(conn net.Conn) *ConnResponseWriter {
	return &ConnResponseWriter{
		conn:       conn,
		header:     make(http.Header),
		statusCode: http.StatusOK, // default to 200 OK
	}
}

// WriteHeader writes the HTTP status code to the client.
func (cw *ConnResponseWriter) WriteHeader(statusCode int) {
	cw.statusCode = statusCode
	cw.wroteHeader = false // Indicate that header has not been written yet
}

// Write sends data to the client connection as part of an HTTP response.
func (cw *ConnResponseWriter) Write(data []byte) (int, error) {
	if !cw.wroteHeader {
		cw.writeHeaders()
	}
	return cw.conn.Write(data)
}

// Header returns the header map that will be sent by WriteHeader.
func (cw *ConnResponseWriter) Header() http.Header {
	return cw.header
}

// writeHeaders writes the headers to the client connection.
func (cw *ConnResponseWriter) writeHeaders() {
	if cw.wroteHeader {
		return // Headers already written
	}

	// Write status line
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", cw.statusCode, http.StatusText(cw.statusCode))
	cw.conn.Write([]byte(statusLine))

	// Write headers
	for key, values := range cw.header {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			cw.conn.Write([]byte(headerLine))
		}
	}

	// End of headers
	cw.conn.Write([]byte("\r\n"))

	cw.wroteHeader = true
}

func serve(wg *sync.WaitGroup) {
	defer wg.Done()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Args.Proxy.Port))
	if err != nil {
		log.Errorf("Error starting TCP server: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Errorf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each connection in a new goroutine
		go handleClient(client)
		//TODO: utilize pooling as guardrail. Add retry mechanism based on timeout
	}
}

func handleClient(client net.Conn) {
	defer client.Close()

	request, err := http.ReadRequest(bufio.NewReader(client))
	if err != nil {
		log.Errorf("Error reading request: %v\n", err)
		return
	}
	// clientReader := bufio.NewReader(client)

	// Read the first line of the request (method and path)
	// requestLine, err := clientReader.ReadString('\n')
	// if err != nil {
	// 	log.Errorf("Error reading request line: %v\n", err)
	// 	return
	// }

	// Parse method and host
	// method, host, ok := parseRequestLine(requestLine)
	// if !ok {
	// 	log.Errorf("Invalid request line: %s\n", requestLine)
	// 	return
	// }

	ps := selectProxy()

	// If method is CONNECT, we're dealing with HTTPS
	if request.Method == "CONNECT" {
		handleTunneling(request.URL.Host, client, ps)
	} else {
		// TODO: Handle regular HTTP requests here, such as accessing HTTP target endpoint
		handleHttpRequest(client, request, ps)
	}

	//TODO: update proxy score based on result
}

func handleHttpRequest(client net.Conn, req *http.Request, ps *types.ProxyServer) {
	var transport http.RoundTripper

	if strings.HasPrefix(ps.Type, "http") {
		proxyURL, err := url.Parse(ps.UrlString())
		if err != nil {
			// reply to client with message "Error parsing proxy URL" and HTTP status code=http.StatusInternalServerError
			log.Errorf("Error parsing proxy URL: %s, %+v", ps.UrlString(), err)
			return
		}
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct)
		if err != nil {
			log.Errorf("Error creating SOCKS5 dialer: %+v", err)
			return
		}
		transport = &http.Transport{
			Dial: dialer.Dial,
		}
	}

	// req.URL.Scheme = "http" // or "https" depending on your needs
	req.RequestURI = "" // Request.RequestURI can't be set in client requests
	req.URL.Host = req.Host
	req.Header.Set("User-Agent", selectUserAgent(ps.UrlString()))

	targetClient := &http.Client{Transport: transport}
	response, err := targetClient.Do(req)
	if err != nil {
		log.Errorf("Error forwarding request: %+v", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading response body: %+v", err)
		return
	}

	// Create our custom ResponseWriter
	cw := NewConnResponseWriter(client)

	for key, values := range response.Header {
		for _, value := range values {
			cw.Header().Add(key, value)
		}
	}
	cw.WriteHeader(response.StatusCode)
	cw.Write(body)
}

func handleTunneling(host string, client net.Conn, ps *types.ProxyServer) {
	//TODO: need to modify the user-agent inside the request header
	if strings.HasPrefix(ps.Type, "http") {
		httpProxy(host, client, ps)
	} else {
		socks5Proxy(host, client, ps)
	}
}

func socks5Proxy(host string, client net.Conn, ps *types.ProxyServer) {
	// Dial the SOCKS5 proxy
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct)
	if err != nil {
		log.Errorf("Error creating SOCKS5 dialer: %v\n", err)
		return
	}

	// Connect to the target host through the SOCKS5 proxy
	server, err := dialer.Dial("tcp", host)
	if err != nil {
		log.Errorf("Error connecting to host through SOCKS5 proxy: %v\n", err)
		return
	}
	defer server.Close()

	// Inform the client that the connection is established
	_, err = client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		log.Errorf("Error sending connection confirmation: %v\n", err)
		return
	}

	// Start tunneling - bidirectional copy
	go io.Copy(server, client)
	io.Copy(client, server)
}

func httpProxy(host string, client net.Conn, ps *types.ProxyServer) {
	// Parse the third-party proxy URL
	proxyUrl, err := url.Parse(ps.UrlString())
	if err != nil {
		log.Errorf("Error parsing proxy URL: %v\n", err)
		return
	}

	// Dial the third-party proxy
	proxyConn, err := net.Dial("tcp", proxyUrl.Host)
	if err != nil {
		log.Errorf("Error connecting to third-party proxy: %v\n", err)
		return
	}
	defer proxyConn.Close()

	// Send a CONNECT request to the third-party proxy
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", host, host)
	_, err = proxyConn.Write([]byte(connectReq))
	if err != nil {
		log.Errorf("Error sending CONNECT request to proxy: %v\n", err)
		return
	}

	// Read the response from the proxy
	proxyReader := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(proxyReader, nil)
	if err != nil {
		log.Errorf("Error reading response from proxy: %v\n", err)
		return
	}
	resp.Body.Close()

	// Check if the proxy connection was successful
	if resp.StatusCode != 200 {
		log.Errorf("Non-200 status code from proxy: %d\n", resp.StatusCode)
		return
	}

	// Inform the original client that the tunnel is established
	_, err = client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		log.Errorf("Error sending connection confirmation to client: %v\n", err)
		return
	}

	// Start tunneling - bidirectional copy
	go io.Copy(proxyConn, client)
	io.Copy(client, proxyConn)
}

func parseRequestLine(requestLine string) (method, host string, ok bool) {
	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 2 {
		return "", "", false
	}
	method = parts[0]
	host = parts[1]
	ok = true
	return
}

func handleRequestAndRedirect(w http.ResponseWriter, req *http.Request) {
	ps := selectProxy()

	if req.Method == "CONNECT" {
		handleConnect(w, req, ps)
		return
	}

	var transport http.RoundTripper

	if strings.HasPrefix(ps.Type, "http") {
		proxyURL, err := url.Parse(ps.UrlString())
		if err != nil {
			http.Error(w, "Error parsing proxy URL", http.StatusInternalServerError)
			return
		}
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct)
		if err != nil {
			log.Errorf("Error creating SOCKS5 dialer: %v", err)
			http.Error(w, "Error forwarding request", http.StatusInternalServerError)
			return
		}
		transport = &http.Transport{
			Dial: dialer.Dial,
		}
	}

	// req.URL.Scheme = "http" // or "https" depending on your needs
	req.URL.Host = req.Host
	req.Header.Set("User-Agent", selectUserAgent(ps.UrlString()))

	client := &http.Client{Transport: transport}
	response, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Error reading response body", http.StatusInternalServerError)
		return
	}

	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	w.Write(body)
}

func handleConnect(w http.ResponseWriter, r *http.Request, ps *types.ProxyServer) {
	// Hijack the connection to get the raw net.Conn
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "HTTP server does not support hijacking", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}

	// Connect to the third-party proxy
	proxyConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port))
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		clientConn.Close()
		return
	}

	// Send CONNECT request to the third-party proxy
	connectRequest := "CONNECT " + r.Host + " HTTP/1.1\r\nHost: " + r.Host + "\r\n"
	for header, values := range r.Header {
		for _, value := range values {
			connectRequest += header + ": " + value + "\r\n"
		}
	}
	connectRequest += "\r\n"
	proxyConn.Write([]byte(connectRequest))

	// Read the response from the proxy. This is simplified and assumes the proxy immediately responds with a success.
	// In a robust implementation, you should parse this response and handle non-success status codes.
	buffer := make([]byte, 4096)
	n, err := proxyConn.Read(buffer)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		clientConn.Close()
		proxyConn.Close()
		return
	}

	//parse the response read into buffer and handle non-success status codes.
	responseString := string(buffer[:n])
	if !strings.HasPrefix(responseString, "HTTP/1.1 200") {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		clientConn.Close()
		proxyConn.Close()
		return
	}

	// Assuming a successful connection, write back the successful HTTP 200 status to the client
	clientConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

	// Now, relay data between the client and the third-party proxy
	go transfer(clientConn, proxyConn)
	go transfer(proxyConn, clientConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

// as a cold start, load the list of proxy servers from database using data.GormDB.
// use types.ProxyServer as data model.
// cache the loaded proxy servers in a string array.
// It has a lifespan of conf.Args.Proxy.MemCacheLifespan (seconds), after which
// fresh data shall be loaded from database again.
func selectProxy() *types.ProxyServer {

	//TODO: value direct:master:rotate proxy weights (from the custom req header?)

	currentTime := time.Now().Unix()
	if len(proxyServersCache) == 0 || currentTime-cacheLastUpdated > int64(conf.Args.Proxy.MemCacheLifespan) {
		proxyServersCache = make([]types.ProxyServer, 0, 16)
		if err := data.GormDB.Find(
			&proxyServersCache,
			"score >= ?",
			conf.Args.Network.RotateProxyScoreThreshold).Error; err != nil {
			log.Fatalf("Failed to load proxy servers from database: %v", err)
		}
		cacheLastUpdated = currentTime
	}

	if len(proxyServersCache) == 0 {
		//TODO: cater fallback_master_proxy config in this case
		return nil
	}

	// Select a random proxy server from the cache
	randomIndex := rand.Intn(len(proxyServersCache))
	return &proxyServersCache[randomIndex]
}

// Select user agent string based on proxyURL.
// Employ the userAgentBinding map of structure map[string]string where the proxyURL is key and user-agent string is its value.
// The map could be empty as lazy-start. In this case, pick a random record using data.GormDB and the model type.UserAgent.
// It has a field UserAgent denoting the user-agent string.
// Then stores and cache the randomly selected string together with the proxyURL in the map.
// We can return the mapped user-agent string without querying the database using GORM next time.
func selectUserAgent(proxyURL string) string {
	userAgent, exists := userAgentBinding[proxyURL]
	if exists {
		return userAgent
	}

	// Assuming data.GormDB is the GORM database instance and type.UserAgent is the model
	var ua types.UserAgent
	if err := data.GormDB.Order("random()").First(&ua).Error; err != nil {
		log.Fatalf("Failed to select random user-agent: %v", err)
		return ""
	}

	userAgentBinding[proxyURL] = ua.UserAgent.String
	return ua.UserAgent.String
}
