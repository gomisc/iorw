package iorw

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"git.corout.in/golibs/errors"
	"git.corout.in/golibs/errors/errgroup"
	"git.corout.in/golibs/slog"
	"gopkg.in/yaml.v2"
)

// ErrUnsupportedFileFormat - не поддерживаемый формат файла
const ErrUnsupportedFileFormat = errors.Const("unsupported file format")

// RemoveLogAll - удаляет переданные файлы и директории с логированием ошибок
func RemoveLogAll(ctx context.Context, paths ...string) {
	log := slog.MustFromContext(ctx)

	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			log.Error("close descriptor", err)
		}
	}
}

// MakeDirs - асинхронно создает директории из списка
func MakeDirs(names ...string) error {
	eg := errgroup.New()

	for _, dir := range names {
		d := dir

		eg.Go(
			func() error {
				if err := os.MkdirAll(d, os.ModePerm); err != nil {
					return errors.Ctx().Str("path", d).Wrap(err, "create directory")
				}

				return nil
			},
		)
	}

	return eg.Wait()
}

// WriteFiles - паралельно записывает данные в файлы
func WriteFiles(src map[string][]byte) error {
	eg := errgroup.New()

	for n, d := range src {
		name, data := n, d

		eg.Go(func() error {
			if err := ioutil.WriteFile(name, data, os.ModePerm); err != nil {
				return errors.Wrapf(err, "write file %s", name)
			}

			return nil
		})
	}

	return eg.Wait()
}

// ReadFileTo - читает файл и декодирует его содержимое в переданный объект
func ReadFileTo(filePath string, result interface{}) (err error) {
	var (
		info os.FileInfo
		fd   *os.File
	)

	info, err = os.Stat(filePath)
	if err != nil {
		return errors.Wrapf(err, "get file stat")
	}

	fd, err = os.OpenFile(filepath.Clean(filePath), os.O_RDONLY, info.Mode())
	if err != nil {
		return errors.Wrapf(err, "open file")
	}

	defer func() {
		if err = fd.Close(); err != nil {
			err = errors.Wrap(err, "close file")
		}
	}()

	buf := &bytes.Buffer{}

	if _, err = io.Copy(buf, fd); err != nil {
		return errors.Wrapf(err, "read file")
	}

	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		if err = yaml.Unmarshal(buf.Bytes(), &result); err != nil {
			return errors.Wrapf(err, "unmarshal yaml")
		}
	case ".json":
		if err = json.Unmarshal(buf.Bytes(), &result); err != nil {
			return errors.Wrapf(err, "unmarshal json")
		}
	default:
		return ErrUnsupportedFileFormat
	}

	return nil
}
