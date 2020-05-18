package store

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
)

var (
	errNotAdvertisable = errors.New("local bind address is not advertisable")
	errNotTCP          = errors.New("local address is not a TCP address")
)

func newTLSTCPTransport(
	certFile string,
	serverKey string,
	bindAddr string,
	advertise net.Addr,
	maxPool int,
	timeout time.Duration,
	logOutput io.Writer,
) (*raft.NetworkTransport, error) {
	cer, err := tls.LoadX509KeyPair(certFile, serverKey)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", bindAddr, config)
	if err != nil {
		return nil, err
	}

	// Create stream
	stream := &tlsStreamLayer{
		advertise: advertise,
		listener:  ln.(*net.TCPListener),
	}

	// Verify that we have a usable advertise address
	addr, ok := stream.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, errNotTCP
	}
	if addr.IP.IsUnspecified() {
		ln.Close()
		return nil, errNotAdvertisable
	}

	if logOutput == nil {
		logOutput = os.Stderr
	}
	// Create the network transport
	trans := raft.NewNetworkTransport(stream, maxPool, timeout, logOutput)
	return trans, nil
}

type tlsStreamLayer struct {
	listener net.Listener
	advertise net.Addr
}

// Dial implements the StreamLayer interface.
func (t *tlsStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", string(address), timeout)
}

// Accept implements the net.Listener interface.
func (t *tlsStreamLayer) Accept() (c net.Conn, err error) {
	return t.listener.Accept()
}

// Close implements the net.Listener interface.
func (t *tlsStreamLayer) Close() (err error) {
	return t.listener.Close()
}

// Addr implements the net.Listener interface.
func (t *tlsStreamLayer) Addr() net.Addr {
	// Use an advertise addr if provided
	if t.advertise != nil {
		return t.advertise
	}
	return t.listener.Addr()
}
