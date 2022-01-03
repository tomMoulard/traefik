package udp

import (
	"errors"
	"io"
	"net"

	"github.com/traefik/traefik/v2/pkg/log"
)

// Proxy is a reverse-proxy implementation of the Handler interface.
type Proxy struct {
	// TODO: maybe optimize by pre-resolving it at proxy creation time
	target net.Addr
}

// NewProxy creates a new Proxy.
func NewProxy(address string) (*Proxy, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	return &Proxy{target: addr}, nil
}

// ServeUDP implements the Handler interface.
func (p *Proxy) ServeUDP(conn *Conn) {
	log.WithoutContext().Debugf("Handling connection from %s", conn.rAddr)

	errChan := make(chan error)
	go func() {
		for {
			buf := make([]byte, maxDatagramSize)
			n, err := conn.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					errChan <- nil
					return
				}

				errChan <- err
				return
			}

			_, err = conn.lConn.WriteTo(buf[:n], p.target)
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	err := <-errChan
	if err != nil {
		log.WithoutContext().Errorf("Error while serving UDP: %v", err)
	}
}
