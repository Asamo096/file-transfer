package xfer

type DeviceID string

type DeviceInfo struct {
	ID       DeviceID `json:"id"`
	Name     string   `json:"name"`
	Addr     string   `json:"addr"`
	Protocol string   `json:"protocol"`
}
