package xfer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DefaultChunkSize = 256 * 1024
	TransferDBPath  = ".transfer_state.db"
)

type TransferState struct {
	FileID       string    `json:"file_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	ChunkSize    int       `json:"chunk_size"`
	TotalChunks  int       `json:"total_chunks"`
	ReceivedChunks int     `json:"received_chunks"`
	FilePath     string    `json:"file_path"`
	PeerID       string    `json:"peer_id"`
	Direction    string    `json:"direction"`
	StartTime    time.Time `json:"start_time"`
	LastUpdate   time.Time `json:"last_update"`
	Status       string    `json:"status"`
	Checksum     string    `json:"checksum"`
}

type Progress struct {
	FileID       string  `json:"file_id"`
	FileName     string  `json:"file_name"`
	TotalSize    int64   `json:"total_size"`
	Transferred  int64   `json:"transferred"`
	Speed        float64 `json:"speed"`
	Progress     float64 `json:"progress"`
	ETA          int     `json:"eta"`
	Status       string  `json:"status"`
}

type TransferManager struct {
	chunkSize    int
	stateFile    string
	transfers    map[string]*TransferState
	activeFiles  map[string]*os.File
	mu           sync.RWMutex
	progressCb   func(*Progress)
}

func NewTransferManager(chunkSize int) *TransferManager {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	return &TransferManager{
		chunkSize:   chunkSize,
		stateFile:   TransferDBPath,
		transfers:   make(map[string]*TransferState),
		activeFiles: make(map[string]*os.File),
	}
}

func (tm *TransferManager) SetProgressCallback(cb func(*Progress)) {
	tm.progressCb = cb
}

func (tm *TransferManager) InitUpload(filePath string, peerID string) (*TransferState, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileID := generateFileID(filePath)
	totalChunks := int((stat.Size() + int64(tm.chunkSize) - 1) / int64(tm.chunkSize))

	state := &TransferState{
		FileID:       fileID,
		FileName:     filepath.Base(filePath),
		FileSize:     stat.Size(),
		ChunkSize:    tm.chunkSize,
		TotalChunks:  totalChunks,
		ReceivedChunks: 0,
		FilePath:     filePath,
		PeerID:       peerID,
		Direction:    "upload",
		StartTime:    time.Now(),
		LastUpdate:   time.Now(),
		Status:       "in_progress",
	}

	tm.mu.Lock()
	tm.transfers[fileID] = state
	tm.mu.Unlock()

	tm.saveState()

	return state, nil
}

func (tm *TransferManager) InitDownload(fileName string, fileSize int64, savePath string, peerID string) (*TransferState, error) {
	fileID := generateFileID(fileName)
	totalChunks := int((fileSize + int64(tm.chunkSize) - 1) / int64(tm.chunkSize))

	exists, received := tm.checkExistingTransfer(fileID)

	state := &TransferState{
		FileID:       fileID,
		FileName:     fileName,
		FileSize:     fileSize,
		ChunkSize:    tm.chunkSize,
		TotalChunks:  totalChunks,
		ReceivedChunks: received,
		FilePath:     filepath.Join(savePath, fileName),
		PeerID:       peerID,
		Direction:    "download",
		StartTime:    time.Now(),
		LastUpdate:   time.Now(),
		Status:       "in_progress",
	}

	if exists {
		state.Status = "resuming"
	}

	tm.mu.Lock()
	tm.transfers[fileID] = state
	tm.mu.Unlock()

	tm.saveState()

	return state, nil
}

func (tm *TransferManager) checkExistingTransfer(fileID string) (bool, int) {
	state, exists := tm.transfers[fileID]
	if exists && state != nil {
		return true, state.ReceivedChunks
	}

	tm.loadState()
	state, exists = tm.transfers[fileID]
	if exists && state != nil {
		return true, state.ReceivedChunks
	}

	return false, 0
}

func (tm *TransferManager) GetChunk(fileID string, chunkIndex int) ([]byte, error) {
	tm.mu.RLock()
	state, exists := tm.transfers[fileID]
	tm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("transfer not found")
	}

	file, err := os.Open(state.FilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	offset := int64(chunkIndex * tm.chunkSize)
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	chunk := make([]byte, tm.chunkSize)
	n, err := file.Read(chunk)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return chunk[:n], nil
}

func (tm *TransferManager) SaveChunk(fileID string, chunkIndex int, data []byte) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	state, exists := tm.transfers[fileID]
	if !exists {
		return fmt.Errorf("transfer not found")
	}

	chunkPath := tm.getChunkPath(fileID, chunkIndex)
	err := os.MkdirAll(filepath.Dir(chunkPath), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(chunkPath, data, 0644)
	if err != nil {
		return err
	}

	state.ReceivedChunks++
	state.LastUpdate = time.Now()

	tm.saveState()

	if tm.progressCb != nil {
		progress := tm.calculateProgress(state)
		tm.progressCb(progress)
	}

	return nil
}

func (tm *TransferManager) CompleteTransfer(fileID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	state, exists := tm.transfers[fileID]
	if !exists {
		return fmt.Errorf("transfer not found")
	}

	if state.ReceivedChunks < state.TotalChunks {
		return fmt.Errorf("transfer incomplete")
	}

	err := tm.assembleFile(state)
	if err != nil {
		state.Status = "failed"
		return err
	}

	state.Status = "completed"
	state.LastUpdate = time.Now()
	tm.saveState()

	tm.cleanupChunks(fileID)

	return nil
}

func (tm *TransferManager) assembleFile(state *TransferState) error {
	destFile, err := os.Create(state.FilePath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	for i := 0; i < state.TotalChunks; i++ {
		chunkPath := tm.getChunkPath(state.FileID, i)
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return err
		}

		_, err = destFile.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tm *TransferManager) cleanupChunks(fileID string) {
	chunkDir := filepath.Join(os.TempDir(), "file-transfer", fileID)
	os.RemoveAll(chunkDir)
}

func (tm *TransferManager) getChunkPath(fileID string, chunkIndex int) string {
	return filepath.Join(os.TempDir(), "file-transfer", fileID, fmt.Sprintf("chunk_%d", chunkIndex))
}

func (tm *TransferManager) calculateProgress(state *TransferState) *Progress {
	transferred := int64(state.ReceivedChunks * state.ChunkSize)
	if transferred > state.FileSize {
		transferred = state.FileSize
	}

	elapsed := time.Since(state.StartTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(transferred) / elapsed
	}

	var eta int
	if speed > 0 {
		remaining := state.FileSize - transferred
		eta = int(remaining / int64(speed))
	}

	progress := float64(0)
	if state.FileSize > 0 {
		progress = float64(transferred) / float64(state.FileSize) * 100
	}

	return &Progress{
		FileID:      state.FileID,
		FileName:    state.FileName,
		TotalSize:   state.FileSize,
		Transferred: transferred,
		Speed:       speed,
		Progress:    progress,
		ETA:         eta,
		Status:      state.Status,
	}
}

func (tm *TransferManager) GetProgress(fileID string) (*Progress, error) {
	tm.mu.RLock()
	state, exists := tm.transfers[fileID]
	tm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("transfer not found")
	}

	return tm.calculateProgress(state), nil
}

func (tm *TransferManager) GetActiveTransfers() []*Progress {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*Progress, 0, len(tm.transfers))
	for _, state := range tm.transfers {
		if state.Status == "in_progress" || state.Status == "resuming" {
			result = append(result, tm.calculateProgress(state))
		}
	}

	return result
}

func (tm *TransferManager) CancelTransfer(fileID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	state, exists := tm.transfers[fileID]
	if !exists {
		return fmt.Errorf("transfer not found")
	}

	state.Status = "cancelled"
	state.LastUpdate = time.Now()
	tm.saveState()

	tm.cleanupChunks(fileID)

	delete(tm.transfers, fileID)

	return nil
}

func (tm *TransferManager) saveState() {
	data, err := json.Marshal(tm.transfers)
	if err != nil {
		return
	}

	os.WriteFile(tm.stateFile, data, 0644)
}

func (tm *TransferManager) loadState() {
	data, err := os.ReadFile(tm.stateFile)
	if err != nil {
		return
	}

	json.Unmarshal(data, &tm.transfers)
}

func (tm *TransferManager) SetChunkSize(size int) {
	if size > 0 {
		tm.chunkSize = size
	}
}

func generateFileID(filePath string) string {
	info, _ := os.Stat(filePath)
	if info != nil {
		return fmt.Sprintf("%s_%d_%d", filepath.Base(filePath), info.Size(), info.ModTime().Unix())
	}
	return fmt.Sprintf("%s_%d", filepath.Base(filePath), time.Now().Unix())
}

type TransferRequest struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	PeerID   string `json:"peer_id"`
}

type TransferResponse struct {
	FileID    string `json:"file_id"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}
