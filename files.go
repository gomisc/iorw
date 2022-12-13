package iorw

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"

	"git.eth4.dev/golibs/errors"
	"git.eth4.dev/golibs/errors/errgroup"
	"git.eth4.dev/golibs/slog"
	"gopkg.in/yaml.v2"

	"git.eth4.dev/golibs/filepaths"
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

// ReadFromFile - читает файл и декодирует его содержимое в переданный объект
func ReadFromFile(filePath string, obj any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "read data from file")
	}

	if err = ReadFromBytes(data, filepath.Ext(filePath), obj); err != nil {
		return errors.Wrap(err, "parse file bytes")
	}

	return nil
}

// ReadFromBytes - читает содержимое бинарного массива в переданный объект
func ReadFromBytes(data []byte, ext string, obj any) error {
	buf := bytes.NewBuffer(data)

	switch ext {
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(buf)
		if err := decoder.Decode(obj); err != nil {
			return errors.Wrapf(err, "unmarshal yaml")
		}
	case ".json":
		decoder := json.NewDecoder(buf)
		if err := decoder.Decode(&obj); err != nil {
			return errors.Wrapf(err, "unmarshal json")
		}
	default:
		decoder := gob.NewDecoder(buf)
		if err := decoder.Decode(&obj); err != nil {
			return errors.Wrapf(err, "decode data from gob format")
		}
	}

	return nil
}

// WriteToFile - записывает содержимое объекта в файл
func WriteToFile(filePath string, obj any) (err error) {
	buf := &bytes.Buffer{}

	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(buf)
		if err = encoder.Encode(&obj); err != nil {
			return errors.Wrap(err, "encode data to yaml")
		}
	case ".json":
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "  ")

		if err = encoder.Encode(&obj); err != nil {
			return errors.Wrap(err, "encode data to json")
		}
	default:
		encoder := gob.NewEncoder(buf)

		if err = encoder.Encode(&obj); err != nil {
			return errors.Wrap(err, "encode data with gob encoder")
		}
	}

	if err = os.WriteFile(filePath, buf.Bytes(), os.ModePerm); err != nil {
		return errors.Wrap(err, "write data to file")
	}

	return nil
}
