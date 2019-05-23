package master

import (
	"encoding/json"
	"io/ioutil"

	"github.com/evilsocket/islazy/log"
)

type NodeConfig struct {
	Address  string `json:"address"`
	CertFile string `json:"credentials,omitempty"`
}

type Config struct {
	Nodes []NodeConfig `json:"nodes"`
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := Config{}

	if data, err := ioutil.ReadFile(configFile); err != nil {
		return nil, err
	} else if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	} else {
		return &cfg, nil
	}
}

func StoreConfig(config *Config, filepath string) error {
	if data, err := json.Marshal(config); err != nil {
		return err
	} else if err = ioutil.WriteFile(filepath, data, 0644); err != nil {
		return err
	}
	return nil
}

func (ms *Service) updateConfig() {
	if ms.configFile == "" {
		return
	}

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	cfg := Config{}
	cfg.Nodes = make([]NodeConfig, 0, len(ms.nodes))

	for _, n := range ms.nodes {
		nc := NodeConfig{Address: n.Name, CertFile: n.CertFile}
		cfg.Nodes = append(cfg.Nodes, nc)
	}

	if err := StoreConfig(&cfg, ms.configFile); err != nil {
		log.Error("cannot save configuration: %v", err)
	}
}
