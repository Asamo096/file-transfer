package dht

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	store      map[string][]byte
	nodeID     string
	peers      map[string]*PeerInfo
	running    bool
	mu         sync.RWMutex
	listener   net.Listener
	connChan   chan net.Conn
	broadcast  *BroadcastService
}

type PeerInfo struct {
	ID        string
	Addr      string
	Port      int
	PublicKey string
	LastSeen  time.Time
}

type BroadcastService struct {
	conn       *net.UDPConn
	nodeID     string
	peers      map[string]*PeerInfo
	mu         sync.RWMutex
	running    bool
}

const (
	ProtocolVersion = "1.0"
	BroadcastPort  = 9999
	DHTPort        = 9998
)

func NewClient(bootstrapNodes []string) (*Client, error) {
	nodeID := generateNodeID()

	client := &Client{
		store:     make(map[string][]byte),
		nodeID:    nodeID,
		peers:     make(map[string]*PeerInfo),
		running:   false,
		connChan:  make(chan net.Conn, 100),
	}

	client.broadcast = NewBroadcastService(nodeID, client.peers)

	return client, nil
}

func NewBroadcastService(nodeID string, peers map[string]*PeerInfo) *BroadcastService {
	return &BroadcastService{
		nodeID: nodeID,
		peers:  peers,
		running: false,
	}
}

func generateNodeID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	c.running = true

	go c.broadcast.Start()
	go c.listenDHT()

	return nil
}

func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false
	c.broadcast.Stop()

	if c.listener != nil {
		c.listener.Close()
	}

	c.store = make(map[string][]byte)
	c.peers = make(map[string]*PeerInfo)

	return nil
}

func (c *Client) GeneratePairingCode() (string, error) {
	b := make([]byte, 4)
	rand.Read(b)
	code := fmt.Sprintf("%06d", rand.Intn(900000)+100000)
	return code, nil
}

func (c *Client) KeyFromCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return fmt.Sprintf("bifrost-%x", hash[:8])
}

func (c *Client) Put(ctx context.Context, code string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.KeyFromCode(code)
	c.store[key] = value
	return nil
}

func (c *Client) Get(ctx context.Context, code string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.KeyFromCode(code)
	if val, ok := c.store[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (c *Client) Provide(ctx context.Context, code string) error {
	c.broadcast.BroadcastPeerInfo()
	return nil
}

func (c *Client) FindProviders(ctx context.Context, code string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var addrs []string
	for _, peer := range c.peers {
		if peer.Addr != "" {
			addrs = append(addrs, fmt.Sprintf("%s:%d", peer.Addr, peer.Port))
		}
	}

	return addrs, nil
}

func (c *Client) JoinWithCode(ctx context.Context, code string) error {
	key := c.KeyFromCode(code)

	c.mu.RLock()
	_, ok := c.store[key]
	c.mu.RUnlock()

	if !ok {
		return fmt.Errorf("code not found in local DHT")
	}

	return nil
}

func (c *Client) GetPeers() []*PeerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*PeerInfo, 0, len(c.peers))
	for _, peer := range c.peers {
		result = append(result, peer)
	}
	return result
}

func (c *Client) GetLocalPeerInfo() *PeerInfo {
	ips := getLocalIPs()
	var primaryIP string
	if len(ips) > 0 {
		primaryIP = ips[0]
	}

	return &PeerInfo{
		ID:        c.nodeID,
		Addr:      primaryIP,
		Port:      DHTPort,
		PublicKey: "",
		LastSeen:  time.Now(),
	}
}

func (c *Client) listenDHT() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", DHTPort))
	if err != nil {
		return err
	}
	c.listener = ln

	for c.running {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go c.handleDHTConnection(conn)
	}

	return nil
}

func (c *Client) handleDHTConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	msg := string(buf[:n])
	parts := strings.Split(msg, "|")

	if len(parts) >= 3 && parts[0] == "PEER_INFO" {
		peer := &PeerInfo{
			ID:   parts[1],
			Addr: parts[2],
		}
		if len(parts) >= 4 {
			fmt.Sscanf(parts[3], "%d", &peer.Port)
		}

		c.mu.Lock()
		c.peers[peer.ID] = peer
		c.mu.Unlock()

		conn.Write([]byte("ACK"))
	}
}

func (c *BroadcastService) Start() error {
	addr := &net.UDPAddr{Port: BroadcastPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	c.conn = conn
	c.running = true

	go c.readLoop()
	go c.broadcastLoop()

	return nil
}

func (c *BroadcastService) Stop() error {
	c.running = false
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

func (c *BroadcastService) readLoop() {
	buf := make([]byte, 4096)
	for c.running {
		if c.conn == nil {
			break
		}
		c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		msg := string(buf[:n])
		parts := strings.Split(msg, "|")

		if len(parts) >= 3 && parts[0] == "PEER_DISCOVER" {
			peerID := parts[1]
			peerIP := addr.IP.String()
			port := DHTPort

			if len(parts) >= 4 {
				fmt.Sscanf(parts[3], "%d", &port)
			}

			c.mu.Lock()
			c.peers[peerID] = &PeerInfo{
				ID:       peerID,
				Addr:     peerIP,
				Port:     port,
				LastSeen: time.Now(),
			}
			c.mu.Unlock()

			response := fmt.Sprintf("PEER_RESPONSE|%s|%s|%d", c.nodeID, getLocalIPs()[0], DHTPort)
			c.conn.WriteToUDP([]byte(response), addr)
		}
	}
}

func (c *BroadcastService) broadcastLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for c.running {
		<-ticker.C
		c.BroadcastPeerInfo()
	}
}

func (c *BroadcastService) BroadcastPeerInfo() {
	if c.conn == nil {
		return
	}

	addrs := getLocalIPs()
	if len(addrs) == 0 {
		return
	}

	msg := fmt.Sprintf("PEER_DISCOVER|%s|%s|%d", c.nodeID, addrs[0], DHTPort)

	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: BroadcastPort,
	}

	c.conn.WriteToUDP([]byte(msg), broadcastAddr)
}

func getLocalIPs() []string {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}

func (c *Client) DiscoverPeers() []*PeerInfo {
	c.broadcast.BroadcastPeerInfo()
	time.Sleep(2 * time.Second)
	return c.GetPeers()
}

func (c *Client) ExchangePeers(targetAddr string) error {
	conn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	peerInfo := c.GetLocalPeerInfo()
	msg := fmt.Sprintf("PEER_INFO|%s|%s|%d", peerInfo.ID, peerInfo.Addr, peerInfo.Port)

	_, err = conn.Write([]byte(msg))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(buf)

	return err
}

func generateRandomCode() string {
	hash := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("%d", binary.BigEndian.Uint32(hash[:4]))
}