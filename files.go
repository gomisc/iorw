package iorw

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"git.corout.in/golibs/errors"
	"git.corout.in/golibs/errors/errgroup"
	"git.corout.in/golibs/slog"
	"gopkg.in/yaml.v2"

	"git.corout.in/golibs/filepaths"
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
			if err := os.WriteFile(name, data, os.ModePerm); err != nil {
				return errors.Wrapf(err, "write file %s", name)
			}

			return nil
		})
	}

	return eg.Wait()
}

// ReadFiles читает файлы в директории с применением опциональным фильтров
// в таблицу имя файла - содержимое
func ReadFiles(path string, filter filepaths.FilesFilter) (map[string][]byte, error) {
	result := make(map[string][]byte)

	filesMap, err := filepaths.MakeFilesMap(path, false, filter)
	if err != nil {
		return nil, errors.Wrap(err, "create files map")
	}

	for name, f := range filesMap {
		if !f.IsDir() {
			var content []byte

			if content, err = os.ReadFile(name); err != nil {
				return nil, errors.Ctx().Str("path", name).Wrap(err, "read file")
			}

			result[name] = content
		}
	}

	return result, nil
}

// ReadFileTo - читает файл и декодирует его содержимое в переданный объект
func ReadFileTo(filePath string, result any) (err error) {
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
		decoder := gob.NewDecoder(buf)

		if err = decoder.Decode(&result); err != nil {
			return errors.Wrapf(err, "decode data from gob format")
		}
	}

	return nil
}

// WriteToFile - записывает содержимое объекта в файл
func WriteToFile(filePath string, obj any) (err error) {
	var (
		fd  *os.File
		buf = &bytes.Buffer{}
	)

	defer func() {
		buf.Reset()

		if err = fd.Close(); err != nil {
			err = errors.Wrap(err, "close file")
		}
	}()

	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(buf)
		if err = encoder.Encode(obj); err != nil {
			return errors.Wrap(err, "encode data to yaml")
		}
	case ".json":
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "  ")

		if err = encoder.Encode(obj); err != nil {
			return errors.Wrap(err, "encode data to json")
		}
	default:
		encoder := gob.NewEncoder(buf)

		if err = encoder.Encode(obj); err != nil {
			return errors.Wrap(err, "encode data with gob encoder")
		}
	}

	fd, err = os.OpenFile(filepath.Clean(filePath), os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "open or create object file")
	}

	if err = fd.Truncate(0); err != nil {
		return errors.Wrap(err, "truncate object file")
	}

	if _, err = fd.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seek object file")
	}

	if _, err = io.Copy(fd, buf); err != nil {
		return errors.Wrap(err, "write data to file")
	}

	return nil
}
