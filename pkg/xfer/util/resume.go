package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ResumeManager struct {
	mu        sync.RWMutex
	chunkSize int
}

type ChunkInfo struct {
	Index    int    `json:"index"`
	Hash     string `json:"hash"`
	Size     int    `json:"size"`
	Complete bool   `json:"complete"`
}

type TransferState struct {
	Filename string      `json:"filename"`
	Size     int64       `json:"size"`
	Chunks   []ChunkInfo `json:"chunks"`
}

func NewResumeManager(chunkSize int) *ResumeManager {
	if chunkSize <= 0 {
		chunkSize = 256 * 1024
	}
	return &ResumeManager{
		chunkSize: chunkSize,
	}
}

func (r *ResumeManager) SplitFile(filePath string) ([]ChunkInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var chunks []ChunkInfo
	buf := make([]byte, r.chunkSize)
	index := 0

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		hash := sha256.Sum256(buf[:n])
		chunks = append(chunks, ChunkInfo{
			Index:    index,
			Hash:     fmt.Sprintf("%x", hash),
			Size:     n,
			Complete: false,
		})
		index++
	}

	return chunks, nil
}

func (r *ResumeManager) GetTransferState(filePath string) (*TransferState, error) {
	statePath := filePath + ".bifrost.state"
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state TransferState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (r *ResumeManager) SaveTransferState(filePath string, state *TransferState) error {
	statePath := filePath + ".bifrost.state"
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

func (r *ResumeManager) CleanupTransferState(filePath string) error {
	statePath := filePath + ".bifrost.state"
	return os.Remove(statePath)
}

func (r *ResumeManager) VerifyChunk(data []byte, expectedHash string) bool {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash) == expectedHash
}