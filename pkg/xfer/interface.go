package xfer

import (
	"context"
)

type Transfer interface {
	Start() (DeviceID, error)
	Stop() error
	DiscoverDevices(ctx context.Context, callback func(DeviceInfo)) error
	SendFile(ctx context.Context, target DeviceID, files []string, progress chan<- Progress) error
	ReceiveFile(ctx context.Context, handler func(TransferRequest) bool) error
	GeneratePairingCode() (string, error)
	JoinWithCode(ctx context.Context, code string) (DeviceID, error)
}

type Discovery interface {
	Start(ctx context.Context) error
	Stop() error
	Discover(callback func(DeviceInfo))
}

type Signaling interface {
	Start(ctx context.Context) error
	Stop() error
	SendCandidates(target DeviceID, candidates []Candidate) error
	ReceiveCandidates(ctx context.Context) (<-chan CandidateMessage, error)
}

type CandidateMessage struct {
	From       DeviceID    `json:"from"`
	Candidates []Candidate `json:"candidates"`
}

type Transport interface {
	Connect(ctx context.Context, candidates []Candidate) error
	Disconnect() error
	Send(ctx context.Context, data []byte) error
	Receive(ctx context.Context) ([]byte, error)
	IsConnected() bool
}

type STUNClient interface {
	GetPublicAddr() (string, error)
	GetAllPublicAddrs() ([]string, error)
	Close() error
}

type DHTClient interface {
	Start(ctx context.Context) error
	Stop() error
	Put(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Provide(ctx context.Context, key string) error
	FindProviders(ctx context.Context, key string) ([]string, error)
}