package xfer

import (
	"sync"
)

type Connection struct {
	mu        sync.RWMutex
	state     ConnectionState
	deviceID  DeviceID
	candidates []Candidate
	remoteID  DeviceID
}

func NewConnection(deviceID DeviceID) *Connection {
	return &Connection{
		state:    StateInit,
		deviceID: deviceID,
	}
}

func (c *Connection) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Connection) SetState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

func (c *Connection) DeviceID() DeviceID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deviceID
}

func (c *Connection) RemoteID() DeviceID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.remoteID
}

func (c *Connection) SetRemoteID(id DeviceID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remoteID = id
}

func (c *Connection) AddCandidates(candidates []Candidate) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.candidates = append(c.candidates, candidates...)
}

func (c *Connection) Candidates() []Candidate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Candidate{}, c.candidates...)
}

func (c *Connection) SortCandidates() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < len(c.candidates); i++ {
		for j := i + 1; j < len(c.candidates); j++ {
			if c.candidates[j].Priority > c.candidates[i].Priority {
				c.candidates[i], c.candidates[j] = c.candidates[j], c.candidates[i]
			}
		}
	}
}