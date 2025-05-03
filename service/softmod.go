package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"github.com/mlctrez/factop/api"
	"github.com/mlctrez/servicego"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var _ Component = (*SoftMod)(nil)

const NewSave = SaveLocation + "/newSave.zip"
const CurrentSoftMod = SaveLocation + "/softmod.zip"

type SoftMod struct {
	servicego.DefaultLogger
	Factorio *Factorio
	Service  *Service
}

func (sm *SoftMod) Start(s *Service) error {
	sm.Service = s
	sm.Logger(s.Log())
	if s.Factorio == nil {
		return errors.New("factorio component not ready")
	}
	sm.Factorio = s.Factorio
	softMod := api.NewSoftModHandler(s.context, s.Nats.conn, sm)
	return s.Nats.Subscribe(softMod.Subject(), softMod.Handler)
}

func (sm *SoftMod) Stop() error {
	return nil
}

func (sm *SoftMod) removeTemp(path string) {
	if removeTemp := os.RemoveAll(path); removeTemp != nil {
		sm.Errorf("error removing temp dir %s", path)
	}
}

func (sm *SoftMod) ApplySoftMod(_ context.Context, req *api.SoftModRequest) (*api.Empty, error) {
	temp, err := os.MkdirTemp("", "softmod")
	if err != nil {
		return nil, err
	}
	defer sm.removeTemp(temp)

	var zipReader *zip.Reader
	reader := bytes.NewReader(req.Payload)
	if zipReader, err = zip.NewReader(reader, int64(reader.Len())); err != nil {
		return nil, err
	}
	for _, f := range zipReader.File {
		if !strings.HasPrefix(f.Name, "save/") {
			return nil, errors.New("invalid softmod file path " + f.Name)
		}
		if err = extractZipItem(temp, zipReader, f.Name); err != nil {
			return nil, err
		}
	}

	if err = os.WriteFile(CurrentSoftMod, req.Payload, 0644); err != nil {
		return nil, err
	}

	if err = sm.Factorio.Stop(); err != nil {
		sm.Errorf("error stopping factorio %s", err)
		return nil, err
	}
	var newSave *os.File
	if newSave, err = os.Create(NewSave); err != nil {
		return nil, err
	}
	zipWriter := zip.NewWriter(newSave)

	var currentSave *zip.ReadCloser
	if currentSave, err = zip.OpenReader(SaveFile); err != nil {
		return nil, err
	}
	for _, f := range currentSave.File {
		// only include files not already in the softmod
		if _, statErr := os.Stat(filepath.Join(temp, f.Name)); os.IsNotExist(statErr) {
			if err = zipWriter.Copy(f); err != nil {
				return nil, err
			}
			if err = zipWriter.Flush(); err != nil {
				return nil, err
			}
		}
	}
	if err = zipWriter.AddFS(os.DirFS(temp)); err != nil {
		return nil, err
	}
	if err = zipWriter.Close(); err != nil {
		return nil, err
	}
	if err = newSave.Close(); err != nil {
		return nil, err
	}
	if err = currentSave.Close(); err != nil {
		return nil, err
	}
	if err = os.Rename(newSave.Name(), SaveFile); err != nil {
		return nil, err
	}

	if !req.SkipRestart {
		if err = sm.Factorio.Restart(); err != nil {
			sm.Errorf("error restarting factorio %s", err)
			return nil, err
		}
	}
	return &api.Empty{}, nil
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
