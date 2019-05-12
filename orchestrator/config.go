package orchestrator

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

type NodeConfig struct {
	Address  string `json:"address"`
	CertFile string `json:"cert_file,omitempty"`
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

func (ms *MuxService) updateConfig() {
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

	if data, err := json.Marshal(&cfg); err != nil {
		log.Errorf("Cannot serialise configuration: %v", err)
	} else if err = ioutil.WriteFile(ms.configFile, data, 0644); err != nil {
		log.Errorf("Cannot save configuration: %v", err)
	}
}
