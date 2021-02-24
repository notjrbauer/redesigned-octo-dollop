package gateway

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Routes          []Route   `yaml:"routes"`
	DefaultResponse Response  `yaml:"default_response"`
	Backends        []Backend `yaml:"backends"`
}

type Route struct {
	PathPrefix string `yaml:"path_prefix"`
	Backend    string `yaml:"backend"`
}

type Response struct {
	Body       string `yaml:"body"`
	StatusCode int    `yaml:"status_code"`
}

type Backend struct {
	Name        string   `yaml:"name"`
	MatchLabels []string `yaml:"match_labels"`
}

func ParseConfig(path string) (Config, error) {
	out := Config{}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return out, errors.Wrap(err, "Opening config")
	}

	err = yaml.NewDecoder(f).Decode(&out)
	if err != nil {
		return out, errors.Wrap(err, "Parsing config")
	}
	return out, nil
}
