package main

import (
	"log"
	"net"
	"net/http"
	"time"
	"testing"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"io/ioutil"
)

type TestHandler struct {}

func (h TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! You have proxied to TestHandler at %s", r.URL.Path)
}

/**
 * 1. Starts a simple HTTP server locally
 * 2. Stubs out the consul lookups, to use the HTTP server as a backend
 * 3. Starts the proxy with the stubbed backend config
 * 4. Makes a request to the proxy, and verifies that it invokes the HTTP server.
 */
func TestConsulProxy(t *testing.T) {
	listener, err := ListenAndServeWithClose(TestHandler{})
	defer listener.Close()
	assertNil(t, err)

	httpServerPort := listener.Addr().(*net.TCPAddr).Port
	proxyPort := getFreePort()

	fmt.Printf("HTTP Server is running on port %v\n", httpServerPort)

	proxied := &ProxiedService{
		ServiceName: "my-test-service",
		LocalIP: "localhost",
		LocalPort: proxyPort,
	}


	config := &ConsulServerConfig {
		Address: "this.is.an.override.address",
	}
	lookup := NewConsulLookup("my-test-service", config)
	lookup.pollIntervalSec = 1
	entry := &consul.ServiceEntry{
		Service: &consul.AgentService {
			Address: "localhost",
			Port: httpServerPort,
		},
	}

	lookup.consulRest = stubConsulRestLookup([]*consul.ServiceEntry{entry}, nil)

	fmt.Println("Starting proxy...")
	proxy := NewConsulProxy(proxied, lookup)
	go proxy.start()
	time.Sleep(1 * time.Second)
	fmt.Println("Proxy started!")

	fmt.Printf("Making HTTP GET request to http://localhost:%v", proxyPort)
	response, err1 := http.Get(fmt.Sprintf("http://localhost:%v", proxyPort))
	assertNil(t, err1)

	fmt.Println("Reading response...")
	bodyBytes, err2 := ioutil.ReadAll(response.Body)
	assertNil(t, err2)

	assertEqual(t, "Hello World! You have proxied to TestHandler at /", string(bodyBytes), "Proxied response")
}

func ListenAndServeWithClose(handler http.Handler) (net.Listener, error) {

	var listener net.Listener
	var addr = ""
	srv := &http.Server{
		Addr: addr,
		Handler: handler,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		err := srv.Serve(tcpKeepAliveListener{
			listener.(*net.TCPListener),
		})
		if err != nil {
			log.Println("HTTP Server Error - ", err)
		}
	}()

	return listener, nil
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func getFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
