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
