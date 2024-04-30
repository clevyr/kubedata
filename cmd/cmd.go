package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/clevyr/kubedb/cmd/dump"
	"github.com/clevyr/kubedb/cmd/exec"
	"github.com/clevyr/kubedb/cmd/portforward"
	"github.com/clevyr/kubedb/cmd/restore"
	"github.com/clevyr/kubedb/cmd/status"
	"github.com/clevyr/kubedb/internal/config"
	"github.com/clevyr/kubedb/internal/config/flags"
	"github.com/clevyr/kubedb/internal/consts"
	"github.com/clevyr/kubedb/internal/notifier"
	"github.com/clevyr/kubedb/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "kubedb",
		Short:             "Painlessly work with databases in Kubernetes.",
		Version:           buildVersion(),
		DisableAutoGenTag: true,

		PersistentPreRunE: preRun,
		SilenceErrors:     true,
	}

	flags.Kubeconfig(cmd)
	flags.Context(cmd)
	flags.Namespace(cmd)
	flags.Dialect(cmd)
	flags.Pod(cmd)
	flags.LogLevel(cmd)
	flags.LogFormat(cmd)
	flags.Healthchecks(cmd)
	flags.Redact(cmd)
	cmd.InitDefaultVersionFlag()

	cmd.AddGroup(
		&cobra.Group{
			ID:    "ro",
			Title: "Read Only Commands:",
		},
		&cobra.Group{
			ID:    "rw",
			Title: "Read/Write Commands:",
		},
	)

	cmd.AddCommand(
		exec.New(),
		dump.New(),
		restore.New(),
		portforward.New(),
		status.New(),
	)

	return cmd
}

func preRun(cmd *cobra.Command, _ []string) error {
	flags.BindKubeconfig(cmd)
	flags.BindLogLevel(cmd)
	flags.BindLogFormat(cmd)
	flags.BindRedact(cmd)
	flags.BindHealthchecks(cmd)

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill, syscall.SIGTERM)
	cmd.PersistentPostRun = func(_ *cobra.Command, _ []string) { cancel() }
	cmd.SetContext(ctx)

	kubeconfig := viper.GetString(consts.KubeconfigKey)
	if kubeconfig == "$HOME" || strings.HasPrefix(kubeconfig, "$HOME"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		kubeconfig = home + kubeconfig[5:]
		viper.Set(consts.KubeconfigKey, kubeconfig)
	}

	config.InitLog(cmd)
	if err := initConfig(); err != nil {
		return err
	}
	config.InitLog(cmd)

	if url := viper.GetString(consts.HealthchecksPingURLKey); url != "" {
		if handler, err := notifier.NewHealthchecks(url); err != nil {
			log.Err(err).Msg("Notifications creation failed")
		} else {
			if err := handler.Started(); err != nil {
				log.Err(err).Msg("Notifications ping start failed")
			}

			util.OnFinalize(func(err error) {
				if err := handler.Finished(err); err != nil {
					log.Err(err).Msg("Notifications ping finished failed")
				}
			})
		}
	}

	return nil
}

func initConfig() error {
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

func buildVersion() string {
	result := util.GetVersion()
	if commit := util.GetCommit(); commit != "" {
		if len(commit) > 8 {
			commit = commit[:8]
		}
		result += " (" + commit + ")"
	}
	return result
}
