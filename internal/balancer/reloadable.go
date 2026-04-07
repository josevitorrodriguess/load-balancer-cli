package balancer

import "sync"

type Reloadable struct {
	mu      sync.RWMutex
	current Balancer
}

func NewReloadable(current Balancer) *Reloadable {
	return &Reloadable{current: current}
}

func (r *Reloadable) Swap(next Balancer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.current = next
}

func (r *Reloadable) NextBackend() (*Backend, error) {
	current := r.get()
	return current.NextBackend()
}

func (r *Reloadable) ReportFailure(url string) error {
	current := r.get()
	return current.ReportFailure(url)
}

func (r *Reloadable) Backends() []Backend {
	current := r.get()
	return current.Backends()
}

func (r *Reloadable) SetBackendAlive(url string, alive bool) error {
	current := r.get()
	return current.SetBackendAlive(url, alive)
}

func (r *Reloadable) ResetFailCount(url string) error {
	current := r.get()
	return current.ResetFailCount(url)
}

func (r *Reloadable) IncrementActiveConnections(url string) error {
	current := r.get()
	return current.IncrementActiveConnections(url)
}

func (r *Reloadable) DecrementActiveConnections(url string) error {
	current := r.get()
	return current.DecrementActiveConnections(url)
}

func (r *Reloadable) get() Balancer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.current
}
