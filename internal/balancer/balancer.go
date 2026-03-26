package balancer

type Balancer interface {
	NextBackend() (*Backend, error)
	ReportFailure(url string) error
	Backends() []Backend
	SetBackendAlive(url string, alive bool) error
	ResetFailCount(url string) error
}
