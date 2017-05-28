package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type config struct {
	Addr        string `yaml:"addr"`
	DbPath      string `yaml:"db_path"`
	UploadsPath string `yaml:"uploads_path"`
}

func loadConfig(filename string) (conf *config, err error) {
	bConf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	conf = &config{}
	if err := yaml.Unmarshal(bConf, conf); err != nil {
		return nil, err
	}

	return conf, nil
}
