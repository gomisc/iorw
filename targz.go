package iorw

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/gomisc/errors.v1"

	"gopkg.in/gomisc/filepaths.v1"
)

// DirTarGz - упаковывает директорию src в dst/name.tar.gz
func DirTarGz(dst, src string, withRelPaths bool, filters ...filepaths.FilesFilter) error {
	content, err := filepaths.MakeFilesMap(src, withRelPaths, filters...)
	if err != nil {
		return errors.Wrap(err, "create source files map")
	}

	return TarGz(dst, src, content)
}

// TarGz - упаковывает архив по содержимому карты [относительный_путь]os.Fileinfo
func TarGz(dst, base string, content map[string]os.FileInfo) error {
	dstfd, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "create temp file")
	}

	gzw := gzip.NewWriter(dstfd)
	tarw := tar.NewWriter(gzw)

	for fp, fi := range content {
		var (
			fd     *os.File
			header *tar.Header
		)

		if fi.IsDir() {
			continue
		}

		fd, err = os.Open(filepath.Join(base, fp)) // nolint: gosec // уже поклинино в джойне
		if err != nil {
			return errors.Wrapf(err, "open %s", fp)
		}

		header, err = tar.FileInfoHeader(fi, "")
		if err != nil {
			return errors.Wrap(err, "make file info header")
		}

		header.Name = fp

		if err = tarw.WriteHeader(header); err != nil {
			return errors.Wrap(err, "write tar header")
		}

		if _, err = io.Copy(tarw, fd); err != nil {
			return errors.Wrapf(err, "write %s to tar", fp)
		}

		if err = fd.Close(); err != nil {
			return errors.Wrapf(err, "close %s", fp)
		}
	}

	return CloseAll(tarw, gzw, dstfd)
}
