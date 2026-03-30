package softmod

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type softmod struct {
	writer *zip.Writer
	base   string
	temp   string
}

func (zu *softmod) Create(name string) (io.Writer, error) {
	zipWriter, err := zu.writer.Create(name)
	if err != nil {
		return nil, err
	}
	if zu.temp == "" {
		return zipWriter, nil
	}
	filePath := filepath.Join(zu.temp, name)
	if err = os.MkdirAll(filepath.Dir(filePath), fs.ModePerm); err != nil {
		return nil, err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	return io.MultiWriter(zipWriter, file), nil
}

//go:embed README.md factop img locale
var files embed.FS

//go:embed controlHeader.lua
var header string

func BuildControlLua() (buf *bytes.Buffer, err error) {
	controlLua := bytes.NewBufferString(header)

	// TODO: these may need ordering
	err = fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "control.lua" || !strings.HasSuffix(path, ".lua") {
			return nil
		}
		// common.lua is a shared helper library required directly by other
		// modules — it has no event handlers and should not be registered.
		if strings.HasSuffix(path, "/common.lua") {
			return nil
		}
		requirePath := strings.ReplaceAll(strings.TrimSuffix(path, ".lua"), "/", ".")
		_, _ = fmt.Fprintf(controlLua, "add_lib(%q)\n", requirePath)
		return nil
	})
	if err != nil {
		controlLua = nil
	}
	return controlLua, err
}

func CreateZip(base string) (buffer *bytes.Buffer, err error) {
	buffer = new(bytes.Buffer)

	zu := &softmod{writer: zip.NewWriter(buffer), base: base}
	if info, statErr := os.Stat("temp"); statErr == nil && info.IsDir() {
		zu.temp = "temp"
		if err = os.RemoveAll(filepath.Join(zu.temp, base)); err != nil {
			return nil, err
		}
	}

	var controlLua *bytes.Buffer
	controlLua, err = BuildControlLua()
	if err != nil {
		return nil, err
	}

	var create io.Writer
	if create, err = zu.Create(base + "/control.lua"); err != nil {
		return nil, err
	}
	if _, err = create.Write(controlLua.Bytes()); err != nil {
		return nil, err
	}
	if closer, ok := create.(io.Closer); ok {
		_ = closer.Close()
	}

	if err = fs.WalkDir(files, ".", zu.walkDirFunc); err != nil {
		return nil, err
	}

	if err = zu.writer.Close(); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (zu *softmod) walkDirFunc(path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	newPath := filepath.Join(zu.base, path)

	create, createErr := zu.Create(newPath)
	if createErr != nil {
		return createErr
	}
	if closer, ok := create.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	open, openErr := files.Open(path)
	if openErr != nil {
		return openErr
	}
	defer func() { _ = open.Close() }()

	if _, copyErr := io.Copy(create, open); copyErr != nil {
		return copyErr
	}
	if flushErr := zu.writer.Flush(); flushErr != nil {
		return flushErr
	}

	return nil
}
