package watch

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

type HandleFunc func()

type Config struct {
	HostRoot   string
	HandleFunc HandleFunc
	Logger     *zap.Logger
}

func New(cfg Config) candy.Watcher {
	return &watcher{
		cfg: cfg,
	}
}

type watcher struct {
	cfg Config
}

func (f *watcher) Run(ctx context.Context) error {
	f.cfg.Logger.Info("starting Watcher", zap.String("HostRoot", f.cfg.HostRoot))
	defer f.cfg.Logger.Info("shutting down Watcher")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(f.cfg.HostRoot); err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Ignoring chmod
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			f.cfg.Logger.Info("watched dir changed", zap.String("dir", f.cfg.HostRoot), zap.String("file", event.Name), zap.Stringer("op", event.Op))
			f.cfg.HandleFunc()
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
