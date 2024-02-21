package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/agux/roprox/internal/cert"
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/types"
	"golang.org/x/net/proxy"
)

var log = logging.Logger

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

func Serve(wg *sync.WaitGroup) {
	defer wg.Done()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Args.Proxy.Port))
	if err != nil {
		log.Errorf("Error starting TCP server: %v\n", err)
		return
	}
	defer listener.Close()

	log.Infof("roprox started successfully.")

	//TODO print out # of healthy public proxies in the backend pool

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

	// Create our custom ResponseWriter
	cw := NewConnResponseWriter(client)

	request, err := http.ReadRequest(bufio.NewReader(client))
	if err != nil {
		emsg := fmt.Sprintf("Error reading request: %v", err)
		log.Error(emsg)
		http.Error(cw, emsg, http.StatusBadRequest)
		return
	}

	ps := selectProxy()

	// If method is CONNECT, we're dealing with HTTPS
	if request.Method == http.MethodConnect {
		handleTunneling(cw, request, client, ps)
	} else {
		// TODO: Handle regular HTTP requests here, such as accessing HTTP target endpoint
		handleHttpRequest(cw, request, ps)
	}

	//TODO: update proxy score based on result
}

func handleHttpRequest(cw http.ResponseWriter, req *http.Request, ps *types.ProxyServer) {
	var transport http.RoundTripper

	if strings.HasPrefix(ps.Type, "http") {
		proxyURL, err := url.Parse(ps.UrlString())
		if err != nil {
			emsg := fmt.Sprintf("Error parsing proxy URL: %s, %+v", ps.UrlString(), err)
			log.Error(emsg)
			http.Error(cw, emsg, http.StatusInternalServerError)
			return
		}
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct)
		if err != nil {
			emsg := fmt.Sprintf("Error creating SOCKS5 dialer: %+v", err)
			log.Error(emsg)
			http.Error(cw, emsg, http.StatusInternalServerError)
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
		emsg := fmt.Sprintf("Error forwarding request: %+v", err)
		log.Error(emsg)
		http.Error(cw, emsg, http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		emsg := fmt.Sprintf("Error reading response body: %+v", err)
		log.Error(emsg)
		http.Error(cw, emsg, http.StatusInternalServerError)
		return
	}

	for key, values := range response.Header {
		for _, value := range values {
			cw.Header().Add(key, value)
		}
	}
	cw.WriteHeader(response.StatusCode)
	cw.Write(body)
}

func handleTunneling(cw http.ResponseWriter, req *http.Request, client net.Conn, ps *types.ProxyServer) {
	//TODO: need to modify the user-agent inside the request header
	newReq := intercept(cw, req, client, ps)

	if strings.HasPrefix(ps.Type, "http") {
		httpProxy(cw, req, client, ps)
	} else {
		socks5Proxy(cw, req, client, ps)
	}
}

func intercept(cw http.ResponseWriter, req *http.Request, client net.Conn, ps *types.ProxyServer) (newReq *http.Request) {
	destHost := req.URL.Host
	// Certificate generation and TLS configuration setup should go here.

	if cert.Generate(commonName string, nil, conf.Args.Proxy.SSLCertificateFolder)

	// Hijack the connection to perform a TLS handshake.
	tlsConn := tls.Server(client, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("TLS handshake error: %v\n", err)
		return
	}

	// Now, you have a decrypted connection with the client.
	// You can read requests, modify them, and write responses.

	// At this point, you would perform your decryption of client requests,
	// modification of headers, and re-encryption before forwarding to serverConn.

	// Similarly, you would decrypt server responses, modify them if needed,
	// and re-encrypt before sending them back to the client.

}

func socks5Proxy(cw http.ResponseWriter, req *http.Request, client net.Conn, ps *types.ProxyServer) {
	endpoint := req.URL.Host
	// Dial the SOCKS5 proxy
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct)
	if err != nil {
		log.Errorf("Error creating SOCKS5 dialer: %v\n", err)
		return
	}

	// Connect to the target endpoint through the SOCKS5 proxy
	server, err := dialer.Dial("tcp", endpoint)
	if err != nil {
		log.Errorf("Error connecting to endpoint through SOCKS5 proxy: %v\n", err)
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

func httpProxy(cw http.ResponseWriter, req *http.Request, client net.Conn, ps *types.ProxyServer) {
	endpoint := req.URL.Host
	// Parse the third-party proxy URL
	proxyUrl, err := url.Parse(ps.UrlString())
	if err != nil {
		log.Errorf("Error parsing proxy URL: %v\n", err)
		return
	}

	// Dial the third-party proxy
	// set a timeout before net.Dial
	// set a timeout before net.Dial
	connTimeout := time.Duration(conf.Args.LocalProbeTimeout) * time.Second
	proxyConn, err := net.DialTimeout("tcp", proxyUrl.Host, connTimeout)
	if err != nil {
		log.Errorf("Error connecting to third-party proxy: %v\n", err)
		return
	}
	defer proxyConn.Close()

	// Send a CONNECT request to the third-party proxy
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", endpoint, endpoint)
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

	go io.Copy(proxyConn, client)
	io.Copy(client, proxyConn)
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
