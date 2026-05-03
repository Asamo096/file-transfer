package handler

import (
	"net/http"

	"file-transfer/backend/service"

	"github.com/gin-gonic/gin"
)

type CryptoHandler struct {
	cryptoService *service.CryptoService
}

func NewCryptoHandler(cryptoService *service.CryptoService) *CryptoHandler {
	return &CryptoHandler{
		cryptoService: cryptoService,
	}
}

func (h *CryptoHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/crypto/key", h.GenerateKeyHandler)
	}
}

func (h *CryptoHandler) GenerateKeyHandler(c *gin.Context) {
	key, err := h.cryptoService.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate key",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key": key,
	})
}