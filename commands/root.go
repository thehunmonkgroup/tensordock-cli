package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/thehunmonkgroup/tensordock-cli/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile           string
	client            *api.Client
	allowInsecureHTTP bool

	rootCmd = &cobra.Command{
		Use:          "tensordock-cli",
		Short:        "TensorDock v2 CLI",
		Version:      api.ClientVersion,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			versionFlag := cmd.Root().Flags().Lookup("version")
			if versionFlag != nil && versionFlag.Changed {
				return nil
			}

			if cmd == configCmd {
				return nil
			}

			serviceURL := viper.GetString("serviceUrl")
			allowInsecureHTTP := viper.GetBool("allowInsecureHTTP")
			debug := viper.GetBool("debug")
			apiToken, err := resolveAPIToken(cmd)
			if err != nil {
				return err
			}
			normalizedServiceURL, err := api.ValidateBaseURL(serviceURL, allowInsecureHTTP)
			if err != nil {
				return err
			}
			authSource, err := describeAPITokenSource(cmd)
			if err != nil {
				return err
			}

			commandDebugf("initializing client service_url=%s auth_source=%s allow_insecure_http=%t", normalizedServiceURL, authSource, allowInsecureHTTP)

			client, err = api.NewClient(normalizedServiceURL, apiToken, debug)
			if err != nil {
				return err
			}
			return nil
		},
	}
)

const rootHelpTemplate = `TensorDock v2 CLI {{.Root.Version}}

{{if .HasParent}}{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{end}}{{if or .Runnable .HasSubCommands}}Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetHelpTemplate(rootHelpTemplate)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.InitDefaultVersionFlag()

	pflags := rootCmd.PersistentFlags()
	pflags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tensordock.yml)")
	pflags.String("apiToken", "", "API token")
	pflags.String("apiTokenEnvVar", "", "Environment variable containing the API token")
	pflags.BoolVar(&allowInsecureHTTP, "allowInsecureHTTP", false, "Allow an insecure http service URL")
	pflags.Bool("debug", false, "Enable debug mode")
	rootCmd.MarkFlagsMutuallyExclusive("apiToken", "apiTokenEnvVar")

	viper.BindPFlag("apiToken", pflags.Lookup("apiToken"))
	viper.BindPFlag("apiTokenEnvVar", pflags.Lookup("apiTokenEnvVar"))
	viper.BindPFlag("allowInsecureHTTP", pflags.Lookup("allowInsecureHTTP"))
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
	viper.SetDefault("allowInsecureHTTP", false)
	commandDebugf("config path selected file=%s", viper.ConfigFileUsed())

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("warning: config file %v not found", viper.ConfigFileUsed())
		commandDebugf("config file missing file=%s err=%v", viper.ConfigFileUsed(), err)
	} else {
		commandDebugf("config file loaded file=%s", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
	commandDebugf("automatic environment resolution enabled")
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

	commandDebugf("reading API token from environment variable name=%s", name)
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		commandDebugf("API token environment variable missing or empty name=%s", name)
		return "", fmt.Errorf("environment variable %q is not set or is empty", name)
	}

	commandDebugf("API token environment variable resolved name=%s", name)
	return value, nil
}
