package main

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	Port int `json:"port"`
}

func LoadConfig(path string) (config *Config, err error) {

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	jsonBytes, err := io.ReadAll(file)
	if err != nil {
		return
	}

	config = &Config{}
	if err = json.Unmarshal(jsonBytes, config); err != nil {
		return nil, err
	}
	return
}
