package paths

import (
	"os"
	"path/filepath"
)

const (
	ConfigDirName  = ".shelp"
	ConfigFileName = "config.json"
)

func GetConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ConfigDirName)
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func EnsureConfigDir() error {
	return os.MkdirAll(GetConfigDir(), 0700)
}
