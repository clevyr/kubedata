package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clevyr/kubedb/internal/consts"
	"github.com/clevyr/kubedb/internal/tui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func LoadViper() error {
	SetViperDefaults()

	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		viper.AddConfigPath(filepath.Join(xdgConfigHome, "kubedb"))
	}
	viper.AddConfigPath(filepath.Join("$HOME", ".config", "kubedb"))
	viper.AddConfigPath(filepath.Join("etc", "kubedb"))

	viper.AutomaticEnv()
	viper.SetEnvPrefix("kubedb")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := viper.ReadInConfig(); err != nil {
		//nolint:errorlint
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error
			log.Debug().Msg("could not find config file")
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("fatal error reading config file: %w", err)
		}
	}

	log.Debug().Str("path", viper.ConfigFileUsed()).Msg("Loaded config file")
	return nil
}

func SetViperDefaults() {
	viper.SetDefault(consts.NamespaceColorKey, map[string]string{
		"[-_]pro?d(uction)?([-_]|$)": string(tui.ColorRed),
	})
}
