package iorw

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/gomisc/errors.v1"
)

// Copy - рекурсивно копирует объект(ы) по пути файловой системы из источника в приемник
func Copy(dst, src string) (err error) {
	info, err := os.Lstat(src)
	if err != nil {
		return errors.Wrapf(err, "get stat filesystem object '%S'", src)
	}

	// копируем симлинк
	if info.Mode()&os.ModeSymlink != 0 {
		return lcopy(dst, src)
	}
	// копируем директорию
	if info.IsDir() {
		return dcopy(dst, src, info)
	}
	// копируем файл
	return fcopy(dst, src, info)
}

// fcopy - копирует файл
func fcopy(dst, src string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return errors.Wrapf(err, "create directory '%s'", dst)
	}

	d, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "create file '%s'", dst)
	}

	if err = os.Chmod(d.Name(), info.Mode()); err != nil {
		return errors.Wrapf(err, "chmod created directory '%s'", d.Name())
	}

	s, err := os.Open(filepath.Clean(src))
	if err != nil {
		return errors.Wrapf(err, "read source file '%s'", src)
	}

	if _, err = io.Copy(d, s); err != nil {
		return errors.Wrapf(err, "copy file content '%s'->'%s'", src, dst)
	}

	if err = CloseAll(s, d); err != nil {
		return errors.Wrap(err, "close descriptors")
	}

	return nil
}

// dcopy - рекурсивно копирует директорию
func dcopy(dst, src string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), info.Mode()); err != nil {
		return errors.Wrapf(err, "create target directory '%s'", dst)
	}

	cont, err := ioutil.ReadDir(src)
	if err != nil {
		return errors.Wrapf(err, "read source directory '%s'", src)
	}

	for _, c := range cont {
		s, d := filepath.Join(src, c.Name()), filepath.Join(dst, c.Name())
		if err = Copy(d, s); err != nil {
			return errors.Wrapf(err, "copy directory '%s'–>'%s'", s, d)
		}
	}

	return nil
}

// lcopy - копирует символические ссылки
func lcopy(dst, src string) error {
	if _, err := os.Readlink(src); err != nil {
		return errors.Wrapf(err, "read symlink '%s'", src)
	}

	if err := os.Symlink(src, dst); err != nil {
		return errors.Wrapf(err, "copy symlink '%s'->'%s'", src, dst)
	}

	return nil
}
