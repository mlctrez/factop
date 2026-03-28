package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
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
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer sm.removeTemp(temp)

	var zipReader *zip.Reader
	reader := bytes.NewReader(data)
	if zipReader, err = zip.NewReader(reader, int64(reader.Len())); err != nil {
		return fmt.Errorf("reading softmod zip: %w", err)
	}
	for _, f := range zipReader.File {
		if !strings.HasPrefix(f.Name, "save/") {
			return errors.New("invalid softmod file path " + f.Name)
		}
		if err = extractZipItem(temp, zipReader, f.Name); err != nil {
			return fmt.Errorf("extracting zip entry %s: %w", f.Name, err)
		}
	}

	if err = os.WriteFile(CurrentSoftMod, data, 0644); err != nil {
		return fmt.Errorf("writing current softmod: %w", err)
	}

	var newSave *os.File
	if newSave, err = os.Create(NewSave); err != nil {
		return fmt.Errorf("creating new save file: %w", err)
	}
	zipWriter := zip.NewWriter(newSave)

	var currentSave *zip.ReadCloser
	if currentSave, err = zip.OpenReader(SaveFile); err != nil {
		return fmt.Errorf("opening current save file: %w", err)
	}
	for _, f := range currentSave.File {
		// only include files not already in the softmod
		if _, statErr := os.Stat(filepath.Join(temp, f.Name)); os.IsNotExist(statErr) {
			if err = zipWriter.Copy(f); err != nil {
				return fmt.Errorf("copying save entry %s: %w", f.Name, err)
			}
			if err = zipWriter.Flush(); err != nil {
				return fmt.Errorf("flushing save entry %s: %w", f.Name, err)
			}
		}
	}
	if err = zipWriter.AddFS(os.DirFS(temp)); err != nil {
		return fmt.Errorf("adding softmod files to save: %w", err)
	}
	if err = zipWriter.Close(); err != nil {
		return fmt.Errorf("closing zip writer: %w", err)
	}
	if err = newSave.Close(); err != nil {
		return fmt.Errorf("closing new save file: %w", err)
	}
	if err = currentSave.Close(); err != nil {
		return fmt.Errorf("closing current save file: %w", err)
	}
	if err = os.Rename(newSave.Name(), SaveFile); err != nil {
		return fmt.Errorf("renaming save file: %w", err)
	}

	return nil
}

func extractZipItem(dir string, zipReader *zip.Reader, name string) (err error) {
	tempFilePath := filepath.Join(dir, name)
	if err = os.MkdirAll(filepath.Dir(tempFilePath), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", name, err)
	}

	var createItem *os.File
	if createItem, err = os.Create(tempFilePath); err != nil {
		return fmt.Errorf("creating file %s: %w", name, err)
	}
	defer func() { _ = createItem.Close() }()

	var zipItem io.ReadCloser
	if zipItem, err = zipReader.Open(name); err != nil {
		return fmt.Errorf("opening zip entry %s: %w", name, err)
	}
	defer func() { _ = zipItem.Close() }()

	if _, err = io.Copy(createItem, zipItem); err != nil {
		return fmt.Errorf("copying zip entry %s: %w", name, err)
	}
	return nil
}
