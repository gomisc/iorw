package iorw

import (
	"context"
	"io"

	"git.eth4.dev/golibs/errors"
	"git.eth4.dev/golibs/slog"
)

// CloseAll - закрывает переданные дескрипторы c возвратом ошибок
func CloseAll(closers ...io.Closer) error {
	var err error

	for _, c := range closers {
		if e := c.Close(); e != nil {
			err = errors.And(err, e)
		}
	}

	return err
}

// CloseLogAll - закрывает переданные дескрипторы c логированием ошибок
func CloseLogAll(ctx context.Context, closers ...io.Closer) {
	log := slog.MustFromContext(ctx)

	for _, c := range closers {
		if err := c.Close(); err != nil {
			log.Error("close descriptor", err)
		}
	}
}
