// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package listeners

import (
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	asl "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	asllib "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl/lib"
)

const TypeASL = "tcp"

// TCP is a listener for establishing client connections on basic TCP protocol.
type ASLServer struct { // [MQTT-4.2.0-1]
	sync.RWMutex
	id                string       // the internal id of the listener
	address           string       // the network address to bind to
	listen            net.Listener // a net.Listener which will listen for new clients
	config            Config       // configuration values for the listener
	log               *slog.Logger // server logger
	end               uint32       // ensure the close methods are only called once
	aslEndpointConfig *asl.EndpointConfig
}

// NewTCP initializes and returns a new TCP listener, listening on an address.
func NewASLListener(config Config, aslEndpointConfig asl.EndpointConfig) *ASLServer {
	aslserver := &ASLServer{
		id:                config.ID,
		address:           config.Address,
		config:            config,
		aslEndpointConfig: &aslEndpointConfig,
	}

	return aslserver
}

func (l *ASLServer) Init(log *slog.Logger) error {
	l.log = log

	var err error

	endpoint := asl.ASLsetupServerEndpoint(l.aslEndpointConfig)

	addr, err := net.ResolveTCPAddr("tcp", l.address)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	aslListener := asllib.ASLListener{
		TcpListener: listener,
		Endpoint:    endpoint,
	}
	l.listen = aslListener

	return err
}

// ID returns the id of the listener.
func (l *ASLServer) ID() string {
	return l.id
}

// Address returns the address of the listener.
func (l *ASLServer) Address() string {
	if l.listen != nil {
		return l.listen.Addr().String()
	}
	return l.address
}

// Protocol returns the address of the listener.
func (l *ASLServer) Protocol() string {
	return "tls"
}

// Serve starts waiting for new TCP connections, and calls the establish
// connection callback for any received.
func (l ASLServer) Serve(establish EstablishFn) {
	for {
		if atomic.LoadUint32(&l.end) == 1 {
			return
		}

		conn, err := l.listen.Accept()
		if err != nil {
			return
		}

		if atomic.LoadUint32(&l.end) == 0 {
			go func() {
				err = establish(l.id, conn)
				if err != nil {
					l.log.Warn("", "error", err)
				}
			}()
		}
	}
}

// Close closes the listener and any client connections.
func (l *ASLServer) Close(closeClients CloseFn) {
	l.Lock()
	defer l.Unlock()

	if atomic.CompareAndSwapUint32(&l.end, 0, 1) {
		closeClients(l.id)
	}

	if l.listen != nil {
		err := l.listen.Close()
		if err != nil {
			return
		}
	}
}
