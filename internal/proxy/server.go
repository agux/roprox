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
		handleTunneling(request, client, ps)
	} else {
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
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
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

	transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if req.URL.Scheme == "" {
		req.URL.Scheme = "https"
	}
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

func handleTunneling(req *http.Request, client net.Conn, ps *types.ProxyServer) {
	var newReq *http.Request
	var newConn *tls.Conn
	var e error

	if newReq, newConn, e = intercept(req, client, ps); e != nil {
		log.Errorf("Error intercepting request: %+v", e)
		return
	}
	defer newConn.Close()

	// Create our custom ResponseWriter
	cw := NewConnResponseWriter(newConn)

	handleHttpRequest(cw, newReq, ps)
}

func intercept(req *http.Request, client net.Conn, ps *types.ProxyServer) (newReq *http.Request, newConn *tls.Conn, e error) {
	hostName := req.URL.Hostname()

	var certificate tls.Certificate
	var found bool
	if certificate, found = certStore[hostName]; !found {
		if certificate, e = cert.LoadOrGenerate(hostName); e != nil {
			return
		}
		certStore[hostName] = certificate
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
	}

	// Inform the original client that the tunnel is established
	_, e = client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if e != nil {
		return
	}

	// Hijack the connection to perform a TLS handshake.
	newConn = tls.Server(client, tlsConfig)
	if err := newConn.Handshake(); err != nil {
		log.Errorf("TLS handshake error: %v\n", err)
		return
	}

	// read request from tlsConn, replace User-Agent header
	if newReq, e = http.ReadRequest(bufio.NewReader(newConn)); e != nil {
		return
	}

	return
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
