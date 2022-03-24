package rules

import (
	"errors"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/listener/mixed"
	"github.com/Dreamacro/clash/tunnel"
	"strconv"
)

var (
	listenerMixPorts = make(map[string]*mixed.Listener)
)

type ListenerPort struct {
	adapter  string
	port     string
	isSource bool
}

func (p *ListenerPort) RuleType() C.RuleType {
	if p.isSource {
		return C.SrcPort
	}
	return C.DstPort
}

func (p *ListenerPort) Match(metadata *C.Metadata) bool {
	if metadata.LocalPort != "0" {
		return metadata.LocalPort == p.port
	}
	return false
}

func (p *ListenerPort) Adapter() string {
	return p.adapter
}

func (p *ListenerPort) Payload() string {
	return p.port
}

func (p *ListenerPort) ShouldResolveIP() bool {
	return false
}

func NewListenerPort(port string, adapter string) (*ListenerPort, error) {
	_, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, errPayload
	}
	startListenerPort(port)
	return &ListenerPort{
		adapter: adapter,
		port:    port,
	}, nil
}

func startListenerPort(port string) (bool, error) {
	if _, ok := listenerMixPorts[port]; ok {
		return false, errors.New("端口已被启用")
	}
	tcpIn := tunnel.TCPIn()
	mixedListener, err := mixed.New(":"+port, tcpIn)
	if err != nil {
		return false, err
	}
	listenerMixPorts[port] = mixedListener
	return true, nil
}
