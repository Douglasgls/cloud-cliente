package forwarding

type ForwardingListener interface {
	OnForwardingStarted(id string)
	OnForwardingStopped(id string)
	OnForwardingError(id string, err error)
}

type NoopListener struct{}

func (n *NoopListener) OnForwardingStarted(id string)        {}
func (n *NoopListener) OnForwardingStopped(id string)        {}
func (n *NoopListener) OnForwardingError(id string, err error) {}
