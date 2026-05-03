package handler

import (
	"net/http"
	"path/filepath"
	"time"

	"file-transfer/backend/service"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	fileService *service.FileService
	receiveOnly bool
}

func NewFileHandler(fileService *service.FileService, receiveOnly bool) *FileHandler {
	return &FileHandler{
		fileService: fileService,
		receiveOnly: receiveOnly,
	}
}

func (h *FileHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/files", h.ListFilesHandler)
		api.POST("/upload", h.UploadHandler)
		api.GET("/download", h.DownloadHandler)
		api.DELETE("/files", h.DeleteHandler)
		api.PATCH("/files", h.RenameHandler)
		api.GET("/read", h.ReadHandler)
		api.POST("/save", h.SaveHandler)
		api.GET("/info", h.InfoHandler)
		api.POST("/mkdir", h.CreateDirHandler)
	}
}

func (h *FileHandler) ListFilesHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	files, err := h.fileService.ListFiles(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var fileList []map[string]any
	for _, f := range files {
		fileList = append(fileList, map[string]any{
			"name":    f.Name,
			"size":    f.Size,
			"isDir":   f.IsDir,
			"modTime": f.ModTime.Format(time.RFC3339),
			"path":    f.Path,
		})
	}

	c.JSON(http.StatusOK, gin.H{"files": fileList})
}

func (h *FileHandler) UploadHandler(c *gin.Context) {
	path := c.PostForm("path")
	if path == "" {
		path = "/"
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if err := h.fileService.UploadFile(path, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file uploaded successfully"})
}

func (h *FileHandler) DownloadHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	file, err := h.fileService.DownloadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(path))
	c.File(file.Name())
}

func (h *FileHandler) DeleteHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	if err := h.fileService.DeleteFile(path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

func (h *FileHandler) RenameHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.OldPath == "" || req.NewPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oldPath and newPath are required"})
		return
	}

	if err := h.fileService.RenameFile(req.OldPath, req.NewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file renamed successfully"})
}

func (h *FileHandler) ReadHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	content, err := h.fileService.ReadTextFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (h *FileHandler) SaveHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	if err := h.fileService.SaveTextFile(req.Path, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file saved successfully"})
}

func (h *FileHandler) InfoHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"receiveOnly": h.receiveOnly,
	})
}

func (h *FileHandler) CreateDirHandler(c *gin.Context) {
	if h.receiveOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "receive only mode"})
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Path == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path and name are required"})
		return
	}

	var newPath string
	if req.Path == "/" {
		newPath = "/" + req.Name
	} else {
		newPath = req.Path + "/" + req.Name
	}

	if err := h.fileService.CreateDir(newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "directory created successfully",
		"path":    newPath,
	})
}