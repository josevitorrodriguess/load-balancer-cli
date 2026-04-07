package config

import (
	"log/slog"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
)

type ApplyFunc func(Config) error

type Reloader struct {
	path      string
	current   Config
	balancer  *balancer.Reloadable
	apply     ApplyFunc
	watcher   *fsnotify.Watcher
	disabled  bool
}

func NewReloader(path string, current Config, lb *balancer.Reloadable, apply ApplyFunc) (*Reloader, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(absolutePath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, err
	}

	return &Reloader{
		path:     filepath.Clean(absolutePath),
		current:  current,
		balancer: lb,
		apply:    apply,
		watcher:  watcher,
	}, nil
}

func (r *Reloader) Start() {
	go func() {
		defer r.watcher.Close()

		for {
			select {
			case event, ok := <-r.watcher.Events:
				if !ok {
					return
				}

				eventPath, err := filepath.Abs(event.Name)
				if err != nil {
					slog.Error("config reload failed", "error", err)
					continue
				}

				if r.disabled || filepath.Clean(eventPath) != r.path {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}

				r.reload()
			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}

				slog.Error("config reload failed", "error", err)
			}
		}
	}()
}

func (r *Reloader) reload() {
	next, err := Load(r.path)
	if err != nil {
		slog.Error("config reload disabled", "error", err)
		r.disabled = true
		return
	}

	if next.Port != r.current.Port {
		slog.Error("config reload disabled", "error", "port change requires restart")
		r.disabled = true
		return
	}

	if err := r.apply(next); err != nil {
		slog.Error("config reload disabled", "error", err)
		r.disabled = true
		return
	}

	r.current = next

	slog.Info("config reloaded", "strategy", next.Strategy, "backends_count", len(next.Backends))
}
