package balancer

type Balancer interface {
	NextBackend() (*Backend, error)
	ReportFailure(url string) error
	Backends() []Backend
	SetBackendAlive(url string, alive bool) error
	ResetFailCount(url string) error
	IncrementActiveConnections(url string) error
	DecrementActiveConnections(url string) error
}
