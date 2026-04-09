package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Resource struct {
	Name           string `yaml:"name"`
	Endpoint       string `yaml:"endpoint"`
	DestinationURL string `yaml:"destination_url"`
}

type Configuration struct {
	Server struct {
		Host       string `yaml:"host"`
		ListenPort string `yaml:"listen_port"`
		Scheme     string `yaml:"scheme"`
	} `yaml:"server"`
	Resources []Resource `yaml:"resources"`
}

func Load(path string) (*Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
