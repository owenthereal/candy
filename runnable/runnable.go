package runnable

import (
	"context"

	"github.com/oklog/run"
)

type Runable interface {
	Run(context.Context) error
}

func RunWithContext(ctx context.Context, runs []Runable) error {
	var g run.Group

	for _, run := range runs {
		run := run

		ctx, cancel := context.WithCancel(ctx)
		g.Add(func() error {
			return run.Run(ctx)
		}, func(err error) {
			cancel()
		})
	}

	return g.Run()
}
