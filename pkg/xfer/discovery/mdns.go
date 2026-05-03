package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"file-transfer/pkg/xfer"
	"github.com/grandcat/zeroconf"
)

const (
	serviceType    = "_bifrost._tcp.local."
	serviceDomain  = "local."
)

type mDNSDiscovery struct {
	server     *zeroconf.Server
	resolver   *zeroconf.Resolver
	callbacks  []func(xfer.DeviceInfo)
	deviceID   xfer.DeviceID
	deviceName string
	listenPort int
	mu         sync.RWMutex
	running    bool
}

func NewmDNSDiscovery(deviceID xfer.DeviceID, deviceName string, listenPort int) xfer.Discovery {
	return &mDNSDiscovery{
		deviceID:   deviceID,
		deviceName: deviceName,
		listenPort: listenPort,
	}
}

func (m *mDNSDiscovery) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.browse(ctx)

	if m.listenPort > 0 {
		return m.advertise()
	}

	return nil
}

func (m *mDNSDiscovery) advertise() error {
	host, err := os.Hostname()
	if err != nil {
		host = "bifrost-device"
	}
	if m.deviceName != "" {
		host = m.deviceName
	}

	server, err := zeroconf.Register(
		host,
		serviceType,
		serviceDomain,
		m.listenPort,
		[]string{fmt.Sprintf("device=%s", m.deviceID), "proto=quic"},
		nil,
	)
	if err != nil {
		return err
	}
	m.server = server
	return nil
}

func (m *mDNSDiscovery) Stop() error {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	if m.server != nil {
		m.server.Shutdown()
		m.server = nil
	}
	if m.resolver != nil {
		m.resolver.Close()
		m.resolver = nil
	}
	return nil
}

func (m *mDNSDiscovery) Discover(callback func(xfer.DeviceInfo)) {
	m.mu.Lock()
	m.callbacks = append(m.callbacks, callback)
	m.mu.Unlock()
}

func (m *mDNSDiscovery) browse(ctx context.Context) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return
	}
	m.resolver = resolver

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			m.handleEntry(entry)
		}
	}(entries)

	go func() {
		err := resolver.Browse(ctx, serviceType, serviceDomain, entries)
		if err != nil {
		}
	}()

	<-ctx.Done()
}

func (m *mDNSDiscovery) handleEntry(entry *zeroconf.ServiceEntry) {
	if entry.Instance == "" {
		return
	}

	var deviceID xfer.DeviceID
	var protocol string
	for _, txt := range entry.Text {
		if strings.HasPrefix(txt, "device=") {
			deviceID = xfer.DeviceID(strings.TrimPrefix(txt, "device="))
		} else if strings.HasPrefix(txt, "proto=") {
			protocol = strings.TrimPrefix(txt, "proto=")
		}
	}

	if deviceID == "" {
		deviceID = xfer.DeviceID(entry.HostName)
	}

	addr := m.getBestAddress(entry)
	if addr == "" {
		return
	}

	if protocol == "" {
		protocol = "quic"
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, callback := range m.callbacks {
		callback(xfer.DeviceInfo{
			ID:       deviceID,
			Name:     entry.Instance,
			Addr:     addr,
			Protocol: protocol,
		})
	}
}

func (m *mDNSDiscovery) getBestAddress(entry *zeroconf.ServiceEntry) string {
	if len(entry.AddrIPv4) > 0 {
		return fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
	}
	if len(entry.AddrIPv6) > 0 {
		return fmt.Sprintf("[%s]:%d", entry.AddrIPv6[0], entry.Port)
	}
	return ""
}