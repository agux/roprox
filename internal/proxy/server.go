package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agux/roprox/internal/cert"
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
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

	if conf.Args.Proxy.BypassTraffic {
		log.Info("roprox started successfully in bypass mode.")
	} else {
		log.Info("roprox started successfully.")
		num := len(proxyCache.GetData())
		if num <= 0 {
			log.Info("currently there's no qualified proxy in the backend. " +
				"Please wait while the scanner is crawling for public proxy resources.")
		}
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
	// var tlsClient *tls.Conn
	// If method is CONNECT, we're dealing with HTTPS. This part is not retryable
	if request.Method == http.MethodConnect {
		if request, _, e = intercept(cw, request, client); e != nil {
			emsg := fmt.Sprintf("Error intercepting request: %+v", e)
			log.Error(emsg)
			http.Error(cw, "", http.StatusBadRequest)
			return
			// if the error (e) denotes TLS handshake error (such as `first record does not look like a TLS handshake`),
			// we need to extract the raw request data ([]byte or string) from e and handle custom protocol.
			// Handle custom protocol based on the error type and raw request data
			// This is a simplified example. Implement according to your protocol specifications.
			// switch rhe := e.(type) {
			// case tls.RecordHeaderError:
			// 	// in this case, we need to try reading raw request content as bytes from the connection underlying the RecordHeaderError
			// 	// Attempt to read raw request content from the connection
			// 	var rawRequest []byte
			// 	rawRequest, e = readFromConnection(tlsClient, 5) // 5 seconds timeout for simplicity
			// 	if e != nil {
			// 		log.Errorf("Failed to read raw request from connection: %v\n", e)
			// 	}
			// 	// concatenate rhe.RecordHeader and rawRequest
			// 	rawRequest = append(rhe.RecordHeader[:], rawRequest...)

			// 	// Handle the raw request based on your custom protocol
			// 	// This is a placeholder for custom protocol handling logic
			// 	// You might want to inspect rawRequest bytes to determine how to proceed
			// 	log.Infof("Received raw request: %s\n", string(rawRequest))

			// 	//forward / write the rawRequest to request.Host and read its response as bytes with timeout
			// 	var destConn net.Conn
			// 	destConn, e = net.DialTimeout("tcp", request.Host, 5*time.Second)
			// 	if e != nil {
			// 		log.Error(e.Error(), http.StatusServiceUnavailable)
			// 		return
			// 	}
			// 	defer destConn.Close()
			// 	destConn.Write(rawRequest)

			// 	// Set a timeout for reading the response
			// 	if err := destConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			// 		log.Errorf("Failed to set read deadline: %v\n", err)
			// 		return
			// 	}

			// 	// Read the response from the destination server
			// 	var responseBuffer bytes.Buffer
			// 	if _, err := io.Copy(&responseBuffer, destConn); err != nil {
			// 		log.Errorf("Failed to read response from destination: %v\n", err)
			// 		return
			// 	}

			// 	log.Infof("Raw response as bytes: %v\n", responseBuffer.Bytes())
			// 	log.Infof("Raw response as string: %s\n", responseBuffer.String())

			// 	// Write the response back to the client
			// 	if _, err := cw.Write(responseBuffer.Bytes()); err != nil {
			// 		log.Errorf("Failed to write response to client: %v\n", err)
			// 	}

			// default:
			// 	log.Errorf("Unhandled error type: %T\n", e)
			// }
		}
	}

	op := func() (e error) {
		var ps *types.ProxyServer
		if !conf.Args.Proxy.BypassTraffic {
			ps = selectProxy()
		}
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

// func handleCustomProtocol() {

// }

func handleHttpRequest(cw *ConnResponseWriter, req *http.Request, ps *types.ProxyServer) (e error) {
	if req.URL != nil && req.URL.Scheme == "" {
		req.URL.Scheme = "https"
	}

	// Request.RequestURI can't be set in client requests
	req.RequestURI = ""
	req.URL.Host = req.Host

	if ps != nil {
		userAgent := conf.Args.Network.DefaultUserAgent
		if uaVal, e := ua.GetUserAgent(ps.UrlString()); e != nil && uaVal == "" {
			log.Warnf("failed to get random user-agent: %+v\nfallback to default User-Agent: %s", e, userAgent)
		} else {
			userAgent = uaVal
		}
		req.Header.Set("User-Agent", userAgent)
	}

	targetClient := &http.Client{
		Timeout: time.Duration(conf.Args.Proxy.BackendProxyTimeout) * time.Second,
	}

	var transport *http.Transport
	if transport, e = network.GetTransport(ps, true); e != nil {
		return
	}
	if ps != nil {
		log.Tracef("relaying HTTP request via proxy [%s]:\n%+v", ps.UrlString(), req)
	}
	targetClient.Transport = transport

	var reqBodyCopy []byte
	if req.Body != nil && conf.Args.Proxy.EnableInspection {
		reqBodyCopy, _ = io.ReadAll(req.Body)
		// After reading the body, it needs to be replaced for the client.Do call
		req.Body = io.NopCloser(bytes.NewBuffer(reqBodyCopy))
	}

	response, err := targetClient.Do(req)
	if err != nil {
		if ps != nil {
			e = errors.Wrapf(err, "failed to relay request to proxy [%s]", ps.UrlString())
		} else {
			e = errors.Wrap(err, "failed to relay request (bypass proxy)")
		}
		log.Warn(e)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		if ps != nil {
			e = errors.Wrapf(err, "Error reading response body from proxy [%s]", ps.UrlString())
		} else {
			e = errors.Wrap(err, "Error reading response body (bypass proxy)")
		}
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

	if conf.Args.Proxy.EnableInspection {
		if err := SaveNetworkTraffic(req, reqBodyCopy, response, body); err != nil {
			log.Warn("failed to save traffic inspection to database: ", err)
		}
	}

	return nil
}

func intercept(cw *ConnResponseWriter, req *http.Request, client net.Conn) (newReq *http.Request, newConn *tls.Conn, e error) {
	hostName := req.URL.Hostname()
	newReq = req
	//TODO: handle authentication tokens
	// authHeader := req.Header.Get("Authorization")
	// if authHeader == "" {
	// 	authHeader = req.Header.Get("Proxy-Authorization")
	// }
	// var authResponse *http.Response
	// var authBody []byte
	// if authHeader != "" {
	// 	if authResponse, e = handleHttpAuthentication(cw, req, client); e != nil {
	// 		return
	// 	}
	// 	// defer authResponse.Body.Close()

	// 	// authBody, e = io.ReadAll(authResponse.Body)
	// 	// if e != nil {
	// 	// 	e = errors.Wrap(e, "Error reading authentication response body")
	// 	// 	log.Warn(e)
	// 	// 	return
	// 	// }
	// }

	var certificate tls.Certificate
	var found bool
	if certificate, found = certStore[hostName]; !found {
		if certificate, e = cert.LoadOrGenerate(hostName); e != nil {
			return
		}
		//FIXME: fatal error: concurrent map writes
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

	// Hijack the connection and try to perform a TLS handshake.
	newConn = tls.Server(client, tlsConfig)
	// swap the connection with intercepted connection
	cw.conn = newConn

	// if authResponse != nil {
	// 	for key, values := range authResponse.Header {
	// 		for _, value := range values {
	// 			cw.Header().Add(key, value)
	// 		}
	// 	}
	// 	cw.WriteHeader(authResponse.StatusCode)
	// 	cw.Write(authBody)
	// }

	if e = newConn.Handshake(); e != nil {
		defer newConn.Close()
		log.Warnf("TLS handshake error: %v\n", e)
		return
	}

	// read request from tlsConn
	if newReq, e = http.ReadRequest(bufio.NewReader(newConn)); e != nil {
		return
	}

	return
}

func handleHttpAuthentication(cw *ConnResponseWriter, r *http.Request, client net.Conn) (response *http.Response, e error) {
	// Log the request details
	logRequestDetails(r)
	// Step 1: Establish a TCP connection to the target server
	var destConn net.Conn
	log.Warnf("client trying to connect host: %s", r.Host)
	destConn, e = net.DialTimeout("tcp", r.Host, 5*time.Second)
	if e != nil {
		log.Error(e.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	cw.WriteHeader(http.StatusOK)

	// // Step 2: Forward the request to the destination server
	// if err := r.Write(destConn); err != nil {
	// 	log.Errorf("Failed to forward request to destination: %v", err)
	// 	e = err
	// 	return
	// }

	// // Read the response from the destination server
	// destConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	// response, e = http.ReadResponse(bufio.NewReader(destConn), r)
	// if e != nil {
	// 	log.Error(e.Error(), http.StatusServiceUnavailable)
	// 	return
	// }

	// // If the destination server responds with a non-200 status, relay that to the client
	// if response.StatusCode != http.StatusOK {
	// 	// Relay the non-200 status to the client, with response.Body
	// 	fmt.Fprintf(client,
	// 		"HTTP/1.1 %d %s\r\n\r\n",
	// 		response.StatusCode,
	// 		http.StatusText(response.StatusCode))
	// 	return
	// }
	return
}

func logRequestDetails(r *http.Request) {
	// Save a copy of the request for logging
	var requestBodyBytes []byte
	if r.Body != nil {
		requestBodyBytes, _ = io.ReadAll(r.Body)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes)) // Reset r.Body to its original state

	// Log the request body
	log.Infof("Request Body: %s\n", string(requestBodyBytes))
}

// func sniffDest(req *http.Request) {
// 	// Extract the target host and port from the request URL
// 	targetHost := req.URL.Host
// 	if targetHost == "" {
// 		targetHost = req.Host
// 	}
// 	if _, _, err := net.SplitHostPort(targetHost); err != nil {
// 		if strings.Contains(err.Error(), "missing port in address") {
// 			// Default to port 80 if not specified
// 			targetHost = net.JoinHostPort(targetHost, "80")
// 		} else {
// 			log.Errorf("Failed to parse host: %v", err)
// 			return
// 		}
// 	}
// 	targetConn, err := net.Dial("tcp", targetHost)
// }

func readFromConnection(conn net.Conn, timeout int) (data []byte, e error) {
	// read all bytes raw data from conn with timeout. Don't assume it's TLS connection.
	conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	buffer := make([]byte, 4096) // Adjust buffer size as needed
	var totalData []byte
	for {
		readBytes, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break // End of data
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break // Timeout reached
			} else {
				return nil, err // Other error occurred
			}
		}
		totalData = append(totalData, buffer[:readBytes]...)
	}
	return totalData, nil
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

// PrettyPrintHeaders formats http.Header into a human-readable string.
func PrettyPrintHeaders(headers http.Header) string {
	var sb strings.Builder
	for name, values := range headers {
		for _, value := range values {
			sb.WriteString(name)
			sb.WriteString(": ")
			sb.WriteString(value)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// SaveNetworkTraffic takes an http.Request, its body, http.Response, and response body,
// maps them to the NetworkTraffic model, and saves it to the database.
func SaveNetworkTraffic(req *http.Request, reqBody []byte, res *http.Response, resBody []byte) (e error) {
	var sourcePort, destinationPort int
	if _, sourcePortStr, e := net.SplitHostPort(req.RemoteAddr); e != nil {
		log.Warnf("failed to parse RemoteAddr: %s", req.RemoteAddr)
		sourcePort = 0
	} else if sourcePort, e = strconv.Atoi(sourcePortStr); e != nil {
		log.Warnf("failed to parse source port: %s", sourcePortStr)
		return e
	}
	if _, destPortStr, e := net.SplitHostPort(req.Host); e != nil {
		log.Warnf("failed to parse Host: %s", req.Host)
		destinationPort = 0
	} else if destinationPort, e = strconv.Atoi(destPortStr); e != nil {
		log.Warnf("failed to parse destination port: %s", destPortStr)
		return e
	}
	networkTraffic := &types.NetworkTraffic{
		Timestamp:             time.Now(),
		SourceIP:              req.RemoteAddr,
		DestinationIP:         req.Host, // This is the host:port
		SourcePort:            sourcePort,
		DestinationPort:       destinationPort,
		Protocol:              req.Proto,
		Method:                req.Method,
		URL:                   req.URL.String(),
		RequestHeaders:        PrettyPrintHeaders(req.Header),
		RequestBody:           reqBody,
		ResponseHeaders:       PrettyPrintHeaders(res.Header),
		ResponseBody:          resBody,
		StatusCode:            uint(res.StatusCode),
		ResponseContentLength: uint(len(resBody)),
		MIMEType:              res.Header.Get("Content-Type"),
	}

	// Save the record to the database
	result := data.GormDB.CreateInBatches(&networkTraffic, 8)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
