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

	var controlLua *bytes.Buffer
	controlLua, err = BuildControlLua()
	if err != nil {
		return nil, err
	}
	err = os.WriteFile("work/control.lua", controlLua.Bytes(), fs.ModePerm)
	if err != nil {
		return nil, err
	}

	var create io.Writer
	if create, err = zu.writer.Create(base + "/control.lua"); err != nil {
		return nil, err
	}
	if _, err = create.Write(controlLua.Bytes()); err != nil {
		return nil, err
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

	create, createErr := zu.writer.Create(newPath)
	if createErr != nil {
		return createErr
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
