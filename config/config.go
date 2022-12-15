package config

import (
	"errors"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type (
	Config struct {
		App struct {
			HTTPPort int    `yaml:"http_port"`
			Host     string `yaml:"host"`
			Folder   string `yaml:"folder"`
			LogLevel string `yaml:"log_level"`
		}
	}
)

// ParseConig - функция получения и обработки конфигурации.
func ParseConfig(path string) (Config, error) {
	var config Config

	filename, err := filepath.Abs(path)
	if err != nil {
		return config, errors.New("get abs file path error")
	}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, errors.New("readFile error")
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return config, errors.New("unmarshal config error")
	}

	return config, nil
}

// Validate - валидация конфигурации.
func (c *Config) Validate() error {
	switch {
	case c.App.Folder == "":
		return errors.New("config app.folder is required")
	case c.App.HTTPPort < 10:
		return errors.New("config http.port is less than 10")
	case c.App.Host == "":
		return errors.New("config app.host is required")
	}

	return nil
}
