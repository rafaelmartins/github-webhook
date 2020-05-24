package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func UnTarRootDir(r io.Reader, outputDir string) error {
	reader := tar.NewReader(r)

	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		dirIndex := strings.Index(filepath.ToSlash(hdr.Name), "/")
		if dirIndex == -1 && !hdr.FileInfo().IsDir() {
			continue
		}

		fn := outputDir
		if dirIndex > 0 {
			fn = filepath.Join(outputDir, hdr.Name[dirIndex:])
		}

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(fn, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
			continue
		}

		if hdr.Linkname != "" {
			if err := os.Symlink(hdr.Linkname, fn); err != nil {
				return err
			}
			continue
		}

		f, err := os.Create(fn)
		if err != nil {
			return err
		}

		if _, err := io.Copy(f, reader); err != nil {
			f.Close()
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

		if err := os.Chmod(fn, os.FileMode(hdr.Mode)); err != nil {
			return err
		}
	}

	return nil
}

func UnTarGzRootDir(r io.Reader, outputDir string) error {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	return UnTarRootDir(reader, outputDir)
}
