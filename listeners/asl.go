// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package listeners

import (
	"crypto/tls"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
)

const TypeASL = "tcp"

// TCP is a listener for establishing client connections on basic TCP protocol.
type ASLListener struct { // [MQTT-4.2.0-1]
	sync.RWMutex
	id      string       // the internal id of the listener
	address string       // the network address to bind to
	listen  net.Listener // a net.Listener which will listen for new clients
	config  Config       // configuration values for the listener
	log     *slog.Logger // server logger
	end     uint32       // ensure the close methods are only called once
}

// NewTCP initializes and returns a new TCP listener, listening on an address.
func NewASLListener(config Config) *ASLListener {
	return &ASLListener{
		id:      config.ID,
		address: config.Address,
		config:  config,
	}
}
func (l *TCP) Init(log *slog.Logger) error {
	l.log = log

	var err error
	if l.config.TLSConfig != nil {
		l.listen, err = tls.Listen("tcp", l.address, l.config.TLSConfig)
	} else {
		l.listen, err = net.Listen("tcp", l.address)
	}

	return err
}

func (l ASLListener) Serve(EstablishFn) {
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

func (l ASLListener) ID() string {

	return l.id
}
func (l ASLListener) Address() string {
	if l.listen != nil {
		return l.listen.Addr().String()
	}
	return l.address
}

func (l ASLListener) Protocol() string {
	return "tls"
}
func (l ASLListener) Close(CloseFn) {
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
