package signaling

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"file-transfer/pkg/xfer"
	"github.com/skip2/go-qrcode"
)

type PairingData struct {
	PublicKey    string         `json:"pk"`
	DeviceID     xfer.DeviceID  `json:"did"`
	Candidates   []string       `json:"candidates"`
	Timestamp    int64          `json:"ts"`
}

type QRCodeSignaling struct {
	deviceID   xfer.DeviceID
	publicKey  string
}

func NewQRCodeSignaling(deviceID xfer.DeviceID) *QRCodeSignaling {
	pk := generatePublicKey()
	return &QRCodeSignaling{
		deviceID:  deviceID,
		publicKey: pk,
	}
}

func generatePublicKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (q *QRCodeSignaling) GenerateQRCode(candidates []string) ([]byte, error) {
	data := PairingData{
		PublicKey:   q.publicKey,
		DeviceID:    q.deviceID,
		Candidates:  candidates,
		Timestamp:   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return qrcode.Encode(string(jsonData), qrcode.Medium, 256)
}

func (q *QRCodeSignaling) ParseQRCode(data []byte) (*PairingData, error) {
	var pairingData PairingData
	err := json.Unmarshal(data, &pairingData)
	if err != nil {
		return nil, err
	}
	return &pairingData, nil
}

func (q *QRCodeSignaling) GenerateManualCode() (string, error) {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", int(b[0])*65536+int(b[1])*256+int(b[2])), nil
}

func (q *QRCodeSignaling) GetPublicKey() string {
	return q.publicKey
}