package handler

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"strings"

	"file-transfer/backend/service"
	"file-transfer/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

type QRCodeHandler struct {
	cryptoService *service.CryptoService
}

func NewQRCodeHandler() *QRCodeHandler {
	return &QRCodeHandler{}
}

func (h *QRCodeHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/qrcode", h.GenerateQRCodeHandler)
		api.GET("/ip", h.GetIPHandler)
	}
}

func (h *QRCodeHandler) GetIPHandler(c *gin.Context) {
	ip := utils.GetLocalIP()
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

func (h *QRCodeHandler) GenerateQRCodeHandler(c *gin.Context) {
	encrypt := c.Query("encrypt") == "true"

	ip := utils.GetLocalIP()
	if ip == "" {
		ip = "127.0.0.1"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	url := fmt.Sprintf("http://%s:%s/", ip, port)

	if encrypt {
		key := generateSecureKey()
		url = fmt.Sprintf("http://%s:%s/#key=%s", ip, port, key)
	}

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	c.Header("Content-Type", "image/png")
	c.Data(http.StatusOK, "image/png", png)
}

func generateSecureKey() string {
	cs := service.NewCryptoService()
	key, _ := cs.GenerateKey()
	return key
}

func (h *QRCodeHandler) GenerateASCIIQRCode(url string) (string, error) {
	pngData, err := qrcode.Encode(url, qrcode.Medium, 20)
	if err != nil {
		return "", err
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			brightness := (r + g + b) / 3

			r2, g2, b2, _ := img.At(x, y+1).RGBA()
			brightness2 := (r2 + g2 + b2) / 3

			if brightness > 0x8000 && brightness2 > 0x8000 {
				result.WriteString("  ")
			} else if brightness <= 0x8000 && brightness2 <= 0x8000 {
				result.WriteString("██")
			} else if brightness <= 0x8000 && brightness2 > 0x8000 {
				result.WriteString("▀▀")
			} else {
				result.WriteString("▄▄")
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}