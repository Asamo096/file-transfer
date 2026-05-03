package stun

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
	"crypto/rand"
)

type Client struct {
	servers []string
}

type STUNMessage struct {
	Type     uint16
	Length   uint16
	Cookie   uint32
	TID      [12]byte
	Attrs    []Attribute
}

type Attribute struct {
	Type  uint16
	Length uint16
	Value []byte
}

const (
	BindingRequest      = 0x0001
	BindingResponse     = 0x0101
	BindingError        = 0x0111
	AttrMappedAddress   = 0x0001
	AttrXORMappedAddr   = 0x0020
	AttrUsername        = 0x0006
	AttrMsgIntegrity    = 0x0008
	AttrError           = 0x0009
	AttrUnknownAttrs    = 0x000A
	AttrRealm           = 0x0014
	AttrNonce           = 0x0015
	AttrXORRelayedAddr  = 0x0016
	AttrReqTrans        = 0x0019
	AttrXORPeerAddr     = 0x0012
	AttrData            = 0x0013
	AttrIceControlled   = 0x8029
	AttrIceControlling  = 0x802A
	AttrPriority        = 0x0024
	AttrUseCandidate    = 0x0025
	AttrSoftware        = 0x8022
	AttrAlternateServer = 0x8023
	AttrFingerprint     = 0x8028
	STUNMagicCookie     = 0x2112A442
)

func NewClient(servers []string) *Client {
	if len(servers) == 0 {
		servers = []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
			"stun2.l.google.com:19302",
			"stun3.l.google.com:19302",
			"stun4.l.google.com:19302",
		}
	}
	return &Client{servers: servers}
}

func (c *Client) GetPublicAddr() (string, error) {
	for _, server := range c.servers {
		addr, err := c.getFromServer(server)
		if err == nil && addr != "" {
			return addr, nil
		}
	}
	return "", fmt.Errorf("failed to get public address from any STUN server")
}

func (c *Client) GetAllPublicAddrs() ([]string, error) {
	var addrs []string
	seen := make(map[string]bool)

	for _, server := range c.servers {
		addr, err := c.getFromServer(server)
		if err == nil && addr != "" && !seen[addr] {
			addrs = append(addrs, addr)
			seen[addr] = true
		}
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("failed to get any public address")
	}
	return addrs, nil
}

func (c *Client) getFromServer(server string) (string, error) {
	conn, err := net.Dial("udp", server)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	req := STUNMessage{
		Type:   BindingRequest,
		Cookie: STUNMagicCookie,
	}
	rand.Read(req.TID[:])

	buf := c.encodeMessage(&req)

	_, err = conn.Write(buf)
	if err != nil {
		return "", err
	}

	readBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(readBuf)
	if err != nil {
		return "", err
	}

	resp, err := c.decodeMessage(readBuf[:n])
	if err != nil {
		return "", err
	}

	for _, attr := range resp.Attrs {
		if attr.Type == AttrXORMappedAddr || attr.Type == AttrMappedAddress {
			return c.parseAddress(&attr, &req), nil
		}
	}

	return "", fmt.Errorf("no address attribute found")
}

func (c *Client) encodeMessage(msg *STUNMessage) []byte {
	buf := make([]byte, 20)
	binary.BigEndian.PutUint16(buf[0:2], msg.Type)
	binary.BigEndian.PutUint16(buf[2:4], 0)
	binary.BigEndian.PutUint32(buf[4:8], msg.Cookie)
	copy(buf[8:20], msg.TID[:])
	return buf
}

func (c *Client) decodeMessage(data []byte) (*STUNMessage, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("invalid stun message")
	}

	msg := &STUNMessage{
		Type:   binary.BigEndian.Uint16(data[0:2]),
		Length: binary.BigEndian.Uint16(data[2:4]),
		Cookie: binary.BigEndian.Uint32(data[4:8]),
	}
	copy(msg.TID[:], data[8:20])

	pos := 20
	for pos < len(data) && pos-20 < int(msg.Length) {
		if pos+4 > len(data) {
			break
		}
		attrType := binary.BigEndian.Uint16(data[pos:pos+2])
		attrLen := binary.BigEndian.Uint16(data[pos+2:pos+4])
		pos += 4

		if pos+int(attrLen) > len(data) {
			break
		}
		attrValue := data[pos:pos+int(attrLen)]
		msg.Attrs = append(msg.Attrs, Attribute{
			Type:  attrType,
			Length: attrLen,
			Value: attrValue,
		})

		padLen := (4 - int(attrLen)%4) % 4
		pos += int(attrLen) + padLen
	}

	return msg, nil
}

func (c *Client) parseAddress(attr *Attribute, req *STUNMessage) string {
	data := attr.Value
	if len(data) < 8 {
		return ""
	}

	family := data[1]
	if family == 0x01 {
		port := binary.BigEndian.Uint16(data[2:4])
		if attr.Type == AttrXORMappedAddr {
			port ^= uint16(req.Cookie >> 16)
		}

		var ip [4]byte
		copy(ip[:], data[4:8])
		if attr.Type == AttrXORMappedAddr {
			xorCookie := uint32ToBytes(req.Cookie)
			for i := 0; i < 4; i++ {
				ip[i] ^= xorCookie[i]
			}
		}

		return fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], port)
	}

	return ""
}

func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func (c *Client) GetNATType() (string, error) {
	for _, server := range c.servers {
		natType, err := c.detectNATType(server)
		if err == nil && natType != "" {
			return natType, nil
		}
	}
	return "unknown", fmt.Errorf("failed to detect NAT type")
}

func (c *Client) detectNATType(server string) (string, error) {
	conn, err := net.Dial("udp", server)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	addr, err := c.getFromServer(server)
	if err != nil {
		return "", err
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	addrStr := fmt.Sprintf("%s:%d", localAddr.IP.String(), localAddr.Port)

	if addr == addrStr {
		return "open", nil
	}

	return "NAT", nil
}

func (c *Client) Close() error {
	return nil
}