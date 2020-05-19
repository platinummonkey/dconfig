package store

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/raft"
)

var (
	errNotAdvertisable = errors.New("local bind address is not advertisable")
	errNotTCP          = errors.New("local address is not a TCP address")
)

// hotSwappingCertificateProvider implements a file-watching certificate provider
// this provides hot-swapping of certificates if certs need to be rotated on a live cluster.
type hotSwappingCertificateProvider struct {
	mu          sync.Mutex
	certFile    string
	keyFile     string
	certificate *tls.Certificate
}

func (p *hotSwappingCertificateProvider) init() (*tls.Certificate, error) {
	cer, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		return nil, err
	}
	return &cer, nil
}

func (p *hotSwappingCertificateProvider) run() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					// this file was modified, attempt to reload
					p.reloadCertificate()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// TODO: something with these errors
			}
		}
	}()

	err = watcher.Add(p.certFile)
	if err != nil {
		return err
	}
	return watcher.Add(p.keyFile)
}

func (p *hotSwappingCertificateProvider) reloadCertificate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cer, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		return err
	}

	p.certificate = &cer
	return nil
}

func (p *hotSwappingCertificateProvider) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.certificate, nil
}

func newTLSTCPTransport(
	certFile string,
	serverKey string,
	bindAddr string,
	advertise net.Addr,
	maxPool int,
	timeout time.Duration,
	logOutput io.Writer,
) (*raft.NetworkTransport, error) {
	certProvider := &hotSwappingCertificateProvider{
		certFile: certFile,
		keyFile:  serverKey,
	}
	_, err := certProvider.init()
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		GetCertificate:           certProvider.getCertificate,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
	}
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
	listener  net.Listener
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
