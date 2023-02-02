package utils

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	directory  string
	tests      bool
	multiStage bool
}

func ReadYml(filename string) (map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	var config map[string]interface{}
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
