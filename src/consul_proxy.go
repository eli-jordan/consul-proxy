package main

import (
	"strconv"
	"net"
	"log"
	"io"
	"os"
)

/**
 * Represents a proxied connection, where the backend is dynamically updated
 * based on the consul service registry.
 */
type ConsulProxy struct {
	// the bind interface, defaults to localhost
	localIp string

	// the port on local host to attach the proxy to
	localPort int

	// handles looking up the currently active set of backend
	// associated with this proxy instance
	lookup    *ConsulLookup
}

/**
 * Creates a new proxy that will listen on the local interface at port 'port'
 * and proxy to the backend specified by 'serviceName'
 *
 * The proxy must be started once created
 */
func NewConsulProxy(service *ProxiedService, consulServer *ConsulServerConfig) *ConsulProxy {
	lookup := NewConsulLookup(service.ServiceName, consulServer)
	lookup.start()

	return &ConsulProxy {
		localIp: service.LocalIP,
		localPort: service.LocalPort,
		lookup: lookup,
	}
}

/**
 * Resolves the local TCP address that the proxy will bind to
 */
func (proxy *ConsulProxy) local() *net.TCPAddr {
	var local = proxy.localIp + ":" + strconv.Itoa(proxy.localPort)
	localAddress, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		panic(err)
	}

	return localAddress
}

func (proxy *ConsulProxy) remote() string {
	remote := proxy.lookup.getEndpoints()[0].String()
	return remote
}

/**
 * Starts up the proxy by listening for TCP connections on the specified local port.
 *
 * Will loop indefinately as new connections are opened
 */
func (proxy *ConsulProxy) start() {
	localAddress := proxy.local()

	listener, err := net.ListenTCP("tcp", localAddress)
	if err != nil {
		log.Fatal("Unable to bind to the local interface", err.Error())
		os.Exit(1)
	}

	for {
		// AcceptTCP will block until a new connection is opened
		log.Println("Now listening on", localAddress, " for service ", proxy.lookup.serviceName)
		localConnection, err := listener.AcceptTCP()
		if err != nil {
			panic(err)
		}

		go proxyConnection(localConnection, proxy.remote())
	}

}

/**
 * Dials the remote address, and proxies any data that is transferred
 * over the connection.
 *
 * Blocks until the connection is closed
 */
func proxyConnection(conn net.Conn, remoteAddress string) {
	backend, err := net.Dial("tcp", remoteAddress)
	defer conn.Close()
	if err != nil {
		log.Fatal("proxy", err.Error())
		return
	}
	defer backend.Close()

	log.Printf("ConsulProxy is proxying to %s", remoteAddress)

	done := make(chan struct{})
	go func() {
		io.Copy(backend, conn)
		backend.(*net.TCPConn).CloseWrite()

		log.Printf("Connection to %s was closed", remoteAddress)
		close(done)
	}()
	io.Copy(conn, backend)
	conn.(*net.TCPConn).CloseWrite()
	<-done
}