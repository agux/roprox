package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/agux/roprox/internal/cert"
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/network"
	"github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/ua"
	"github.com/agux/roprox/internal/util"
	"github.com/avast/retry-go"
	"github.com/pkg/errors"
)

var log = logging.Logger
var masterProxy *types.ProxyServer

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

	log.Info("roprox started successfully.")

	num := len(proxyCache.GetData())
	if num <= 0 {
		log.Info("currently there's no qualified proxy in the backend. " +
			"Please wait while the scanner is crawling for public proxy resources.")
	}

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Errorf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each connection in a new goroutine
		go handleClient(client)
		//TODO: utilize pooling as guardrail.
	}
}

func handleClient(client net.Conn) {
	// Create our custom ResponseWriter
	cw := NewConnResponseWriter(client)
	defer cw.conn.Close()

	request, err := http.ReadRequest(bufio.NewReader(client))
	if err != nil {
		emsg := fmt.Sprintf("Error reading request: %+v", err)
		log.Error(emsg)
		http.Error(cw, emsg, http.StatusBadRequest)
		return
	}

	var e error
	// If method is CONNECT, we're dealing with HTTPS. This part is not retryable
	if request.Method == http.MethodConnect {
		if request, client, e = intercept(request, client); e != nil {
			emsg := fmt.Sprintf("Error intercepting request: %+v", err)
			log.Error(emsg)
			http.Error(cw, emsg, http.StatusBadRequest)
		}

		// swap the connection with intercepted connection
		cw.conn = client
	}

	op := func() (e error) {
		ps := selectProxy()
		e = handleHttpRequest(cw, request, ps)
		network.UpdateProxyScore(ps, e == nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(conf.Args.Proxy.MaxRetryDuration)*time.Second)
	defer cancel()
	if e := retry.Do(
		op,
		retry.Delay(0),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	); e != nil {
		http.Error(cw, e.Error(), http.StatusInternalServerError)
	}
}

func handleHttpRequest(cw *ConnResponseWriter, req *http.Request, ps *types.ProxyServer) (e error) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = "https"
	}

	req.RequestURI = "" // Request.RequestURI can't be set in client requests
	req.URL.Host = req.Host

	userAgent := conf.Args.Network.DefaultUserAgent
	if uaVal, e := ua.GetUserAgent(ps.UrlString()); e != nil {
		log.Warnf("failed to get random user-agent: %+v\nfallback to default User-Agent: %s", e, userAgent)
	} else {
		userAgent = uaVal
	}
	req.Header.Set("User-Agent", userAgent)

	var transport *http.Transport
	if transport, e = network.GetTransport(ps, true); e != nil {
		return
	}
	targetClient := &http.Client{
		Timeout:   time.Duration(conf.Args.Proxy.BackendProxyTimeout) * time.Second,
		Transport: transport,
	}

	log.Tracef("relaying HTTP request via proxy [%s]:\n%+v", ps.UrlString(), req)
	response, err := targetClient.Do(req)
	if err != nil {
		e = errors.Wrapf(err, "failed to relay request to proxy [%s]", ps.UrlString())
		log.Warn(e)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		e = errors.Wrapf(err, "Error reading response body from proxy [%s]", ps.UrlString())
		log.Warn(e)
		return err
	}

	for key, values := range response.Header {
		for _, value := range values {
			cw.Header().Add(key, value)
		}
	}
	cw.WriteHeader(response.StatusCode)
	cw.Write(body)

	return nil
}

func intercept(req *http.Request, client net.Conn) (newReq *http.Request, newConn *tls.Conn, e error) {
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
	if e = newConn.Handshake(); e != nil {
		log.Errorf("TLS handshake error: %v\n", e)
		return
	}

	// read request from tlsConn
	if newReq, e = http.ReadRequest(bufio.NewReader(newConn)); e != nil {
		return
	}

	return
}

// randomly select a proxy from the cache
func selectProxy() *types.ProxyServer {

	cache := proxyCache.GetData()

	if len(cache) > 0 {
		//TODO: consider (per request) direct:master:rotate proxy weights (from the custom req header?)
		// Select a random proxy server from the cache
		return &cache[rand.Intn(len(cache))]
	}

	if !conf.Args.Proxy.FallbackMasterProxy {
		return nil
	}

	log.Warnf("no qualified proxy at the moment. falling back to master proxy: %s", conf.Args.Network.MasterProxyAddr)
	return util.GetMasterProxy()
}
