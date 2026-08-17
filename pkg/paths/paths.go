package paths

import (
	"os"
	"path/filepath"
)

const (
	ConfigDirName  = ".shelp"
	ConfigFileName = "config.json"
	ConfigDirEnv   = "SHELP_CONFIG_DIR"
)

func GetConfigDir() string {
	if dir := os.Getenv(ConfigDirEnv); dir != "" {
		return dir
	}
	return filepath.Join(homeDir(), ConfigDirName)
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func EnsureConfigDir() error {
	return os.MkdirAll(GetConfigDir(), 0700)
}

// os.UserHomeDir only fails when $HOME is unset, in which case the empty
// fallback keeps the path relative rather than pointing at the filesystem root.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}
