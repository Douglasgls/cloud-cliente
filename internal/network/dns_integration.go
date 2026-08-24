package network

type DnsSystemIntegration interface {
	Enable(listenAddr string) error
	Disable() error
}
