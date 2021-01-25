package fswatch

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

type Config struct {
	DomainDir string
}

func New(cfg Config) candy.Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &fsWatcher{
		Config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

type fsWatcher struct {
	Config Config
	ctx    context.Context
	cancel context.CancelFunc
}

func (f *fsWatcher) Watch(h candy.WatcherHandleFunc) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go f.loop(watcher, h)

	if err := watcher.Add(f.Config.DomainDir); err != nil {
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

			candy.Log().Info("watched dir changed", zap.String("dir", f.Config.DomainDir), zap.String("file", event.Name), zap.Stringer("op", event.Op))
			h()
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			candy.Log().Error("error watching dir", zap.String("dir", f.Config.DomainDir), zap.Error(err))
		}
	}
}

func (f *fsWatcher) Shutdown() error {
	defer f.cancel()
	return nil
}
