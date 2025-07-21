package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const NewSave = SaveLocation + "/newSave.zip"
const CurrentSoftMod = SaveLocation + "/softmod.zip"

type SoftMod struct {
	slog.Logger
}

func (sm *SoftMod) removeTemp(path string) {
	if removeTemp := os.RemoveAll(path); removeTemp != nil {
		sm.Error("error removing temp dir", "dir", path)
	}
}

func (sm *SoftMod) Apply(data []byte) error {
	temp, err := os.MkdirTemp("", "softmod")
	if err != nil {
		return err
	}
	defer sm.removeTemp(temp)

	var zipReader *zip.Reader
	reader := bytes.NewReader(data)
	if zipReader, err = zip.NewReader(reader, int64(reader.Len())); err != nil {
		return err
	}
	for _, f := range zipReader.File {
		if !strings.HasPrefix(f.Name, "save/") {
			return errors.New("invalid softmod file path " + f.Name)
		}
		if err = extractZipItem(temp, zipReader, f.Name); err != nil {
			return err
		}
	}

	if err = os.WriteFile(CurrentSoftMod, data, 0644); err != nil {
		return err
	}

	var newSave *os.File
	if newSave, err = os.Create(NewSave); err != nil {
		return err
	}
	zipWriter := zip.NewWriter(newSave)

	var currentSave *zip.ReadCloser
	if currentSave, err = zip.OpenReader(SaveFile); err != nil {
		return err
	}
	for _, f := range currentSave.File {
		// only include files not already in the softmod
		if _, statErr := os.Stat(filepath.Join(temp, f.Name)); os.IsNotExist(statErr) {
			if err = zipWriter.Copy(f); err != nil {
				return err
			}
			if err = zipWriter.Flush(); err != nil {
				return err
			}
		}
	}
	if err = zipWriter.AddFS(os.DirFS(temp)); err != nil {
		return err
	}
	if err = zipWriter.Close(); err != nil {
		return err
	}
	if err = newSave.Close(); err != nil {
		return err
	}
	if err = currentSave.Close(); err != nil {
		return err
	}
	if err = os.Rename(newSave.Name(), SaveFile); err != nil {
		return err
	}

	return nil
}

func extractZipItem(dir string, zipReader *zip.Reader, name string) (err error) {
	tempFilePath := filepath.Join(dir, name)
	if err = os.MkdirAll(filepath.Dir(tempFilePath), 0755); err != nil {
		return err
	}

	var createItem *os.File
	if createItem, err = os.Create(tempFilePath); err != nil {
		return err
	}
	defer func() { _ = createItem.Close() }()

	var zipItem io.ReadCloser
	if zipItem, err = zipReader.Open(name); err != nil {
		return err
	}
	defer func() { _ = zipItem.Close() }()

	_, err = io.Copy(createItem, zipItem)
	return err
}
