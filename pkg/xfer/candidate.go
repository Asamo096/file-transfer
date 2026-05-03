package xfer

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"file-transfer/pkg/xfer/stun"
)

type CandidateType int

const (
	CandidateTypeHost CandidateType = iota
	CandidateTypeSrflx
	CandidateTypeRelay
	CandidateTypePrflx
)

type Candidate struct {
	Type      CandidateType
	Addr      string
	Port      int
	Priority  int
	Component int
	Foundation string
}

type CandidateGatherer struct {
	localIP     string
	stunServer string
	candidates  []*Candidate
	mu         sync.RWMutex
}

func NewCandidateGatherer(localIP string, stunServer string) *CandidateGatherer {
	return &CandidateGatherer{
		localIP:    localIP,
		stunServer: stunServer,
		candidates: make([]*Candidate, 0),
	}
}

func (c *CandidateGatherer) Gather() ([]*Candidate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.candidates = make([]*Candidate, 0)

	c.addCandidate(CandidateTypeHost, c.localIP, 0, 100)

	stunCandidates, err := c.gatherStunCandidates()
	if err == nil {
		c.candidates = append(c.candidates, stunCandidates...)
	}

	return c.candidates, nil
}

func (c *CandidateGatherer) addCandidate(cType CandidateType, addr string, port int, priority int) {
	foundation := fmt.Sprintf("%d", cType)
	c.candidates = append(c.candidates, &Candidate{
		Type:      cType,
		Addr:      addr,
		Port:      port,
		Priority:  priority,
		Component: 1,
		Foundation: foundation,
	})
}

func (c *CandidateGatherer) gatherStunCandidates() ([]*Candidate, error) {
	if c.stunServer == "" {
		c.stunServer = "stun.l.google.com:19302"
	}

	stunClient := stun.NewClient([]string{c.stunServer})
	addr, err := stunClient.GetPublicAddr()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid STUN address")
	}

	var port int
	fmt.Sscanf(parts[1], "%d", &port)

	candidate := &Candidate{
		Type:      CandidateTypeSrflx,
		Addr:      parts[0],
		Port:      port,
		Priority:  90,
		Component: 1,
		Foundation: "srflx",
	}

	return []*Candidate{candidate}, nil
}

func (c *CandidateGatherer) GetLocalAddress() string {
	return c.localIP
}

func (c *CandidateGatherer) GetCandidates() []*Candidate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Candidate, len(c.candidates))
	copy(result, c.candidates)
	return result
}

type CandidateExchange struct {
	localCandidates []*Candidate
	remoteCandidates []*Candidate
	selectedPair   *CandidatePair
	mu             sync.RWMutex
}

type CandidatePair struct {
	Local  *Candidate
	Remote *Candidate
}

func NewCandidateExchange() *CandidateExchange {
	return &CandidateExchange{
		localCandidates: make([]*Candidate, 0),
		remoteCandidates: make([]*Candidate, 0),
	}
}

func (ce *CandidateExchange) SetLocalCandidates(candidates []*Candidate) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.localCandidates = candidates
}

func (ce *CandidateExchange) SetRemoteCandidates(candidates []*Candidate) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.remoteCandidates = candidates
}

func (ce *CandidateExchange) SelectPair() *CandidatePair {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	var bestPair *CandidatePair
	var bestPriority int

	for _, local := range ce.localCandidates {
		for _, remote := range ce.remoteCandidates {
			priority := calculatePairPriority(local, remote)
			if priority > bestPriority {
				bestPriority = priority
				bestPair = &CandidatePair{Local: local, Remote: remote}
			}
		}
	}

	ce.selectedPair = bestPair
	return bestPair
}

func calculatePairPriority(local, remote *Candidate) int {
	localWeight := 0
	remoteWeight := 0

	switch local.Type {
	case CandidateTypeHost:
		localWeight = 100
	case CandidateTypeSrflx:
		localWeight = 90
	case CandidateTypePrflx:
		localWeight = 80
	case CandidateTypeRelay:
		localWeight = 70
	}

	switch remote.Type {
	case CandidateTypeHost:
		remoteWeight = 100
	case CandidateTypeSrflx:
		remoteWeight = 90
	case CandidateTypePrflx:
		remoteWeight = 80
	case CandidateTypeRelay:
		remoteWeight = 70
	}

	return localWeight*remoteWeight + local.Priority + remote.Priority
}

func (ce *CandidateExchange) GetSelectedPair() *CandidatePair {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.selectedPair
}

func CandidateToString(c *Candidate) string {
	return fmt.Sprintf("%s:%d", c.Addr, c.Port)
}

func ParseCandidate(s string) (*Candidate, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid candidate format")
	}

	var port int
	fmt.Sscanf(parts[1], "%d", &port)

	return &Candidate{
		Addr: parts[0],
		Port: port,
	}, nil
}

type PeerAddress struct {
	ID        string
	IP        string
	Port      int
	DeviceName string
	Type      string
}

func (p *PeerAddress) String() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

func GatherAllAddresses() ([]string, error) {
	var addrs []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs2, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs2 {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {
				if ip.To4() != nil {
					addrs = append(addrs, fmt.Sprintf("ipv4:%s", ip.String()))
				} else {
					addrs = append(addrs, fmt.Sprintf("ipv6:[%s]", ip.String()))
				}
			}
		}
	}

	stunClient := stun.NewClient(nil)
	if pubAddr, err := stunClient.GetPublicAddr(); err == nil {
		addrs = append(addrs, fmt.Sprintf("public:%s", pubAddr))
	}

	return addrs, nil
}

type ConnectionState int

const (
	StateInit ConnectionState = iota
	StateGathering
	StateConnecting
	StateConnected
	StateDisconnected
	StateFailed
)

func (s ConnectionState) String() string {
	switch s {
	case StateInit:
		return "Initializing"
	case StateGathering:
		return "Gathering Candidates"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateDisconnected:
		return "Disconnected"
	case StateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

type Peer struct {
	ID         string
	Name       string
	Addresses  []string
	LastSeen   time.Time
	Connection ConnectionState
}