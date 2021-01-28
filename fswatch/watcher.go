package fswatch

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

type Config struct {
	HostRoot string
	Logger   *zap.Logger
}

func New(cfg Config) candy.Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &fsWatcher{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

type fsWatcher struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
}

func (f *fsWatcher) Watch(h candy.WatcherHandleFunc) error {
	f.cfg.Logger.Info("starting watcher", zap.Reflect("cfg", f.cfg))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go f.loop(watcher, h)

	if err := watcher.Add(f.cfg.HostRoot); err != nil {
		return err
	}

	<-f.ctx.Done()
	return f.ctx.Err()
}

func (f *fsWatcher) loop(watcher *fsnotify.Watcher, h candy.WatcherHandleFunc) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Ignoring chmod
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			f.cfg.Logger.Info("watched dir changed", zap.String("dir", f.cfg.HostRoot), zap.String("file", event.Name), zap.Stringer("op", event.Op))
			h()
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			f.cfg.Logger.Error("error watching dir", zap.String("dir", f.cfg.HostRoot), zap.Error(err))
		}
	}
}

func (f *fsWatcher) Shutdown() error {
	f.cfg.Logger.Info("shutting down watcher")

	defer f.cancel()
	return nil
}
