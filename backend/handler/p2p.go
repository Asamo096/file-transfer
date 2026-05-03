package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"file-transfer/pkg/xfer"
	"file-transfer/pkg/xfer/dht"

	"github.com/gin-gonic/gin"
)

type P2PHandler struct {
	dhtClient      *dht.Client
	candidateGath *xfer.CandidateGatherer
	transferMgr   *xfer.TransferManager
	nodeID        string
}

func NewP2PHandler() *P2PHandler {
	dhtClient, _ := dht.NewClient(nil)
	localIP := getLocalIP()
	candidateGath := xfer.NewCandidateGatherer(localIP, "")

	return &P2PHandler{
		dhtClient:      dhtClient,
		candidateGath: candidateGath,
		transferMgr:   xfer.NewTransferManager(256 * 1024),
		nodeID:        generateNodeID(),
	}
}

func (h *P2PHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/p2p")
	{
		api.GET("/devices", h.ListDevicesHandler)
		api.GET("/discover", h.DiscoverHandler)
		api.POST("/pairing", h.GeneratePairingHandler)
		api.POST("/pairing/join", h.JoinPairingHandler)
		api.GET("/addresses", h.GetAddressesHandler)
		api.GET("/stun/address", h.GetSTUNAddressHandler)
		api.GET("/transfers", h.ListTransfersHandler)
		api.GET("/transfers/:id", h.GetTransferHandler)
		api.DELETE("/transfers/:id", h.CancelTransferHandler)
	}
}

func (h *P2PHandler) ListDevicesHandler(c *gin.Context) {
	peers := h.dhtClient.GetPeers()

	var devices []map[string]interface{}
	for _, peer := range peers {
		devices = append(devices, map[string]interface{}{
			"id":       peer.ID,
			"ip":       peer.Addr,
			"port":     peer.Port,
			"lastSeen": peer.LastSeen.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

func (h *P2PHandler) DiscoverHandler(c *gin.Context) {
	peers := h.dhtClient.DiscoverPeers()

	var devices []map[string]interface{}
	for _, peer := range peers {
		devices = append(devices, map[string]interface{}{
			"id":       peer.ID,
			"ip":       peer.Addr,
			"port":     peer.Port,
			"lastSeen": peer.LastSeen.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"count":   len(devices),
		"status":  "discovered",
	})
}

func (h *P2PHandler) GeneratePairingHandler(c *gin.Context) {
	code, err := h.dhtClient.GeneratePairingCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate pairing code"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addrs, _ := xfer.GatherAllAddresses()

	h.dhtClient.Put(ctx, code, []byte(fmt.Sprintf("%s|%s", h.nodeID, strings.Join(addrs, ","))))

	c.JSON(http.StatusOK, gin.H{
		"code":     code,
		"nodeId":   h.nodeID,
		"addresses": addrs,
		"expires":  time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		"qrData":  fmt.Sprintf("%s|%s", code, h.nodeID),
	})
}

func (h *P2PHandler) JoinPairingHandler(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if len(req.Code) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pairing code"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := h.dhtClient.Get(ctx, req.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pairing code not found"})
		return
	}

	peerInfo := string(data)
	parts := strings.Split(peerInfo, "|")

	if len(parts) < 2 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid peer data"})
		return
	}

	peerID := parts[0]
	peerAddrs := strings.Split(parts[1], ",")

	c.JSON(http.StatusOK, gin.H{
		"peerId":   peerID,
		"addresses": peerAddrs,
		"status":  "connected",
	})
}

func (h *P2PHandler) GetAddressesHandler(c *gin.Context) {
	addrs, err := xfer.GatherAllAddresses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to gather addresses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"addresses": addrs,
		"nodeId":   h.nodeID,
	})
}

func (h *P2PHandler) GetSTUNAddressHandler(c *gin.Context) {
	stunClient := NewSTUNClient(nil)
	publicAddr, err := stunClient.GetPublicAddr()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"address": nil,
			"error":  err.Error(),
		})
		return
	}

	natType, _ := stunClient.GetNATType()

	c.JSON(http.StatusOK, gin.H{
		"address": publicAddr,
		"natType": natType,
	})
}

func (h *P2PHandler) ListTransfersHandler(c *gin.Context) {
	transfers := h.transferMgr.GetActiveTransfers()
	c.JSON(http.StatusOK, gin.H{
		"transfers": transfers,
		"count":     len(transfers),
	})
}

func (h *P2PHandler) GetTransferHandler(c *gin.Context) {
	fileID := c.Param("id")
	progress, err := h.transferMgr.GetProgress(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transfer not found"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *P2PHandler) CancelTransferHandler(c *gin.Context) {
	fileID := c.Param("id")
	err := h.transferMgr.CancelTransfer(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transfer cancelled"})
}

type STUNClient struct {
	servers []string
}

func NewSTUNClient(servers []string) *STUNClient {
	if len(servers) == 0 {
		servers = []string{
			"stun.l.google.com:19302",
		}
	}
	return &STUNClient{servers: servers}
}

func (c *STUNClient) GetPublicAddr() (string, error) {
	for _, server := range c.servers {
		addr, err := c.getFromServer(server)
		if err == nil && addr != "" {
			return addr, nil
		}
	}
	return "", fmt.Errorf("failed to get public address")
}

func (c *STUNClient) getFromServer(server string) (string, error) {
	conn, err := net.Dial("udp", server)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	reqData := buildSTUNBindingRequest()
	_, err = conn.Write(reqData)
	if err != nil {
		return "", err
	}

	readBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(readBuf)
	if err != nil {
		return "", err
	}

	return parseXORMappedAddr(readBuf[:n])
}

func (c *STUNClient) GetNATType() (string, error) {
	return "NAT", nil
}

func buildSTUNBindingRequest() []byte {
	msg := make([]byte, 20)
	msg[0], msg[1] = 0x00, 0x01
	msg[2], msg[3] = 0x00, 0x00
	msg[4], msg[5], msg[6], msg[7] = 0x21, 0x12, 0xA4, 0x42

	copy(msg[8:20], generateTID())

	return msg
}

func parseXORMappedAddr(data []byte) (string, error) {
	if len(data) < 20 {
		return "", fmt.Errorf("response too short")
	}

	msgLen := int(data[2])<<8 | int(data[3])
	if msgLen < 12 || len(data) < 20+msgLen {
		return "", fmt.Errorf("invalid message length")
	}

	pos := 20
	for pos < len(data) && pos-20 < msgLen {
		if pos+4 > len(data) {
			break
		}

		attrType := int(data[pos])<<8 | int(data[pos+1])
		attrLen := int(data[pos+2])<<8 | int(data[pos+3])
		pos += 4

		if attrType == 0x0020 && attrLen >= 8 {
			if pos+8 > len(data) {
				break
			}

			port := int(data[pos+2])<<8|int(data[pos+3]) ^ 0x2112

			ip := fmt.Sprintf("%d.%d.%d.%d",
				int(data[pos+4])^0x21,
				int(data[pos+5])^0x12,
				int(data[pos+6])^0xA4,
				int(data[pos+7])^0x42,
			)

			return fmt.Sprintf("%s:%d", ip, port), nil
		}

		padLen := (4 - attrLen%4) % 4
		pos += attrLen + padLen
	}

	return "", fmt.Errorf("XOR-MAPPED-ADDRESS not found")
}

func generateTID() []byte {
	tid := make([]byte, 12)
	for i := range tid {
		tid[i] = byte(time.Now().UnixNano() >> (i * 8) & 0xFF)
	}
	return tid
}

func getLocalIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}

func generateNodeID() string {
	return fmt.Sprintf("node-%d", time.Now().UnixNano())
}

func init() {
	_ = json.Marshal
}