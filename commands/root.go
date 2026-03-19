package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/caguiclajmg/tensordock-cli/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	client  *api.Client

	rootCmd = &cobra.Command{
		Use:          "tensordock-cli",
		Short:        "TensorDock v2 CLI",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd == configCmd {
				return nil
			}

			serviceURL := viper.GetString("serviceUrl")
			debug := viper.GetBool("debug")
			apiToken, err := resolveAPIToken(cmd)
			if err != nil {
				return err
			}

			client = api.NewClient(serviceURL, apiToken, debug)
			return nil
		},
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	pflags := rootCmd.PersistentFlags()
	pflags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tensordock.yml)")
	pflags.String("apiToken", "", "API token")
	pflags.String("apiTokenEnvVar", "", "Environment variable containing the API token")
	pflags.Bool("debug", false, "Enable debug mode")
	rootCmd.MarkFlagsMutuallyExclusive("apiToken", "apiTokenEnvVar")

	viper.BindPFlag("apiToken", pflags.Lookup("apiToken"))
	viper.BindPFlag("apiTokenEnvVar", pflags.Lookup("apiTokenEnvVar"))
	viper.BindPFlag("debug", pflags.Lookup("debug"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.SetConfigFile(filepath.Join(home, ".tensordock.yml"))
	}

	viper.SetDefault("serviceUrl", "https://dashboard.tensordock.com/api/v2")

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("warning: config file %v not found", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
}

func resolveAPIToken(cmd *cobra.Command) (string, error) {
	if cmd != nil {
		apiToken, err := cmd.Flags().GetString("apiToken")
		if err != nil {
			return "", err
		}
		if cmd.Flags().Changed("apiToken") {
			if apiToken == "" {
				return "", fmt.Errorf("api token cannot be empty")
			}
			return apiToken, nil
		}

		apiTokenEnvVar, err := cmd.Flags().GetString("apiTokenEnvVar")
		if err != nil {
			return "", err
		}
		if cmd.Flags().Changed("apiTokenEnvVar") {
			return readTokenFromEnvVar(apiTokenEnvVar)
		}
	}

	if apiToken := viper.GetString("apiToken"); apiToken != "" {
		return apiToken, nil
	}

	if apiTokenEnvVar := viper.GetString("apiTokenEnvVar"); apiTokenEnvVar != "" {
		return readTokenFromEnvVar(apiTokenEnvVar)
	}

	return "", fmt.Errorf("api token is not configured; use --apiToken, --apiTokenEnvVar, or run `tensordock-cli config`")
}

func readTokenFromEnvVar(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("api token environment variable name cannot be empty")
	}

	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %q is not set or is empty", name)
	}

	return value, nil
}
