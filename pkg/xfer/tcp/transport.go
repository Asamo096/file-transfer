package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"
)

type Transport struct {
	conn     net.Conn
	listener net.Listener
	running  bool
	mu       sync.RWMutex
}

func NewTransport() *Transport {
	return &Transport{}
}

func (t *Transport) Listen(port int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	t.listener = listener
	t.running = true

	go t.acceptLoop()

	return nil
}

func (t *Transport) ListenTLS(port int, cert tls.Certificate) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), config)
	if err != nil {
		return err
	}

	t.listener = listener
	t.running = true

	go t.acceptLoop()

	return nil
}

func (t *Transport) acceptLoop() {
	for {
		t.mu.RLock()
		running := t.running
		listener := t.listener
		t.mu.RUnlock()

		if !running || listener == nil {
			return
		}

		conn, err := listener.Accept()
		if err != nil {
			return
		}

		t.mu.Lock()
		if t.conn != nil {
			t.conn.Close()
		}
		t.conn = conn
		t.mu.Unlock()
	}
}

func (t *Transport) Connect(ctx context.Context, addr string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.Close()
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}

	t.conn = conn
	t.running = true

	return nil
}

func (t *Transport) ConnectTLS(ctx context.Context, addr string, serverName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.Close()
	}

	config := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, config)
	if err != nil {
		return err
	}

	t.conn = conn
	t.running = true

	return nil
}

func (t *Transport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}

	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}

	t.running = false

	return nil
}

func (t *Transport) Send(ctx context.Context, data []byte) error {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	_, err := conn.Write(data)
	return err
}

func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (t *Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.conn != nil
}