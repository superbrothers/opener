package main

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func LoadOpenerOptionsFromConfig(configPath string, o *OpenerOptions) error {
	if configPath == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		configPath = filepath.Join(dir, ".config", "opener", "config.yaml")
		if _, err := os.Stat(configPath); err != nil {
			// The config file does not exist in the default path.
			return nil
		}
	} else {
		if _, err := os.Stat(configPath); err != nil {
			return err
		}
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, o); err != nil {
		return err
	}

	return nil
}
