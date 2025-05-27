package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func CheckLatest() (err error) {
	var releases *Releases
	if releases, err = LatestReleases(); err != nil {
		return err
	}
	parent := "/opt/factorio"
	path := filepath.Join(parent, releases.Stable.Headless)
	if _, err = os.Stat(path); os.IsNotExist(err) {
		if err = DownloadStableHeadless(parent); err != nil {
			return err
		}
	}

	return nil
}

// DownloadStableHeadless downloads the latest stable release to parent/version
func DownloadStableHeadless(parent string) error {
	lr, err := LatestReleases()
	if err != nil {
		return err
	}

	command := exec.Command("tar", "xJf", "-", "--strip-components=1")
	downloadDir := filepath.Join(parent, lr.Stable.Headless)
	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return err
	}
	command.Dir = filepath.Join(parent, lr.Stable.Headless)

	resp, err := http.Get(fmt.Sprintf("https://www.factorio.com/get-download/%s/headless/linux64", lr.Stable.Headless))
	if err != nil {
		return err
	}
	command.Stdin = resp.Body
	defer func() { _ = resp.Body.Close() }()
	output, err := command.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}

// LatestReleases returns the latest releases of Factorio
func LatestReleases() (*Releases, error) {
	var releases Releases

	resp, err := http.Get("https://factorio.com/api/latest-releases")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("non-200 status code")
	}

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return nil, err
	}
	return &releases, nil
}

// Releases is the structure of the JSON at https://factorio.com/api/latest-releases
type Releases struct {
	Experimental struct {
		Alpha     string `json:"alpha"`
		Demo      string `json:"demo"`
		Expansion string `json:"expansion"`
		Headless  string `json:"headless"`
	} `json:"experimental"`
	Stable struct {
		Alpha     string `json:"alpha"`
		Demo      string `json:"demo"`
		Expansion string `json:"expansion"`
		Headless  string `json:"headless"`
	} `json:"stable"`
}
