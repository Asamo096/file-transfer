package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

type QUICTransport struct {
	conn      quic.Connection
	listener  *quic.Listener
	running   bool
	mu        sync.RWMutex
	tlsConfig *tls.Config
}

func NewQUICTransport(tlsConfig *tls.Config) *QUICTransport {
	if tlsConfig == nil {
		tlsConfig = generateTLSConfig()
	}
	return &QUICTransport{
		tlsConfig: tlsConfig,
	}
}

func generateTLSConfig() *tls.Config {
	cert, err := tls.X509KeyPair([]byte{}, []byte{})
	if err != nil {
		return &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
}

func (t *QUICTransport) Listen(port int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", port), t.tlsConfig, nil)
	if err != nil {
		return err
	}

	t.listener = listener
	t.running = true

	go t.acceptLoop()

	return nil
}

func (t *QUICTransport) acceptLoop() {
	for {
		t.mu.RLock()
		running := t.running
		listener := t.listener
		t.mu.RUnlock()

		if !running || listener == nil {
			return
		}

		conn, err := listener.Accept(context.Background())
		if err != nil {
			return
		}

		t.mu.Lock()
		if t.conn != nil {
			t.conn.CloseWithError(0, "new connection")
		}
		t.conn = conn
		t.mu.Unlock()
	}
}

func (t *QUICTransport) Connect(ctx context.Context, addr string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.CloseWithError(0, "connecting to new address")
	}

	conn, err := quic.DialAddr(ctx, addr, t.tlsConfig, nil)
	if err != nil {
		return err
	}

	t.conn = conn
	t.running = true

	return nil
}

func (t *QUICTransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.CloseWithError(0, "disconnect")
		t.conn = nil
	}

	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}

	t.running = false

	return nil
}

func (t *QUICTransport) Send(ctx context.Context, data []byte) error {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	_, err = stream.Write(data)
	return err
}

func (t *QUICTransport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (t *QUICTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.conn != nil
}

func (t *QUICTransport) LocalAddr() net.Addr {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.listener != nil {
		return t.listener.Addr()
	}
	if t.conn != nil {
		return t.conn.LocalAddr()
	}
	return nil
}