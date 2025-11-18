package ice

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"
)

const IceURL = "https://osisaf-hl.met.no/ice_edge/ice_edge_latest.json"
const CacheFile = "data/ice_latest.geojson"

type IceLoader struct {
	Client *http.Client
}

func NewIceLoader() *IceLoader {
	return &IceLoader{
		Client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (l *IceLoader) DownloadIceData() error {
	resp, err := l.Client.Get(IceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to download ice data: " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tmp interface{}
	if json.Unmarshal(data, &tmp) != nil {
		return errors.New("invalid JSON in ice data")
	}

	os.MkdirAll("data", 0755)

	return os.WriteFile(CacheFile, data, 0644)
}

func (l *IceLoader) LoadCached() ([]byte, error) {
	return os.ReadFile(CacheFile)
}
