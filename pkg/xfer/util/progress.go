package util

import (
	"fmt"
	"sync"
	"time"

	"file-transfer/pkg/xfer"
)

type ProgressTracker struct {
	mu          sync.RWMutex
	totalSize   int64
	transferred int64
	startTime   time.Time
	lastUpdate  time.Time
	lastBytes   int64
	speed       float64
	filename    string
	status      string
}

func NewProgressTracker(filename string, totalSize int64) *ProgressTracker {
	return &ProgressTracker{
		filename:    filename,
		totalSize:   totalSize,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		status:      "transferring",
	}
}

func (p *ProgressTracker) Update(transferred int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.transferred = transferred
	now := time.Now()
	elapsed := now.Sub(p.lastUpdate).Seconds()

	if elapsed > 0 {
		bytesDelta := transferred - p.lastBytes
		p.speed = float64(bytesDelta) / elapsed
	}

	p.lastUpdate = now
	p.lastBytes = transferred
}

func (p *ProgressTracker) Progress() xfer.Progress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	percentage := 0.0
	if p.totalSize > 0 {
		percentage = (float64(p.transferred) / float64(p.totalSize)) * 100
	}

	elapsed := time.Now().Sub(p.startTime).Seconds()

	return xfer.Progress{
		Filename:     p.filename,
		TotalSize:    p.totalSize,
		Transferred:  p.transferred,
		Speed:        p.speed,
		Percentage:   percentage,
		Status:       p.status,
	}
}

func (p *ProgressTracker) SetStatus(status string) {
	p.mu.Lock()
	p.status = status
	p.mu.Unlock()
}

func (p *ProgressTracker) Complete() {
	p.mu.Lock()
	p.transferred = p.totalSize
	p.status = "completed"
	p.mu.Unlock()
}

func (p *ProgressTracker) Failed(err error) {
	p.mu.Lock()
	p.status = "failed"
	p.mu.Unlock()
}

func FormatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
}

func FormatSpeed(bytesPerSecond float64) string {
	if bytesPerSecond < 1024 {
		return fmt.Sprintf("%d B/s", int(bytesPerSecond))
	}
	if bytesPerSecond < 1024*1024 {
		return fmt.Sprintf("%.2f KB/s", bytesPerSecond/1024)
	}
	return fmt.Sprintf("%.2f MB/s", bytesPerSecond/(1024*1024))
}