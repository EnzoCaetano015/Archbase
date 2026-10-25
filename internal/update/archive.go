package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

// Releases contain exactly one regular executable. Never extract archive paths.
func extractBinary(archive []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, fmt.Errorf("invalid ZIP archive: %w", err)
		}
		if len(reader.File) != 1 || reader.File[0].Name != "arc.exe" || !reader.File[0].Mode().IsRegular() {
			return nil, errors.New("ZIP must contain only a regular arc.exe")
		}
		file, err := reader.File[0].Open()
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return readBinary(file)
	}
	compressed, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	header, err := reader.Next()
	if err != nil {
		return nil, fmt.Errorf("invalid TAR archive: %w", err)
	}
	if header.Name != "arc" || header.Typeflag != tar.TypeReg {
		return nil, errors.New("TAR must contain only a regular arc")
	}
	binary, err := readBinary(reader)
	if err != nil {
		return nil, err
	}
	if _, err = reader.Next(); err != io.EOF {
		return nil, errors.New("TAR contains extra or invalid entries")
	}
	// Reach the gzip trailer so a damaged compressed stream is rejected too.
	if n, readErr := io.Copy(io.Discard, io.LimitReader(compressed, 1025)); readErr != nil {
		return nil, readErr
	} else if n != 0 {
		return nil, errors.New("TAR has trailing data")
	}
	return binary, nil
}

func readBinary(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxArchive+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxArchive {
		return nil, errors.New("release executable is empty or exceeds size limit")
	}
	return data, nil
}
