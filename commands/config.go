package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/caguiclajmg/tensordock-cli/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Set API token configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiToken, err := cmd.Flags().GetString("apiToken")
			if err != nil {
				return err
			}

			apiTokenEnvVar, err := cmd.Flags().GetString("apiTokenEnvVar")
			if err != nil {
				return err
			}

			serviceURL, err := cmd.Flags().GetString("serviceUrl")
			if err != nil {
				return err
			}
			allowInsecureHTTP, err := cmd.Flags().GetBool("allowInsecureHTTP")
			if err != nil {
				return err
			}

			if apiToken == "" && apiTokenEnvVar == "" {
				return fmt.Errorf("either --apiToken or --apiTokenEnvVar must be provided")
			}
			normalizedServiceURL, err := api.ValidateBaseURL(serviceURL, allowInsecureHTTP)
			if err != nil {
				return err
			}

			commandDebugf("config update requested service_url=%s api_token_set=%t api_token_env_var=%q allow_insecure_http=%t", normalizedServiceURL, apiToken != "", apiTokenEnvVar, allowInsecureHTTP)

			if apiToken != "" {
				if err := confirmAuthReplacement("apiTokenEnvVar"); err != nil {
					return err
				}
				viper.Set("apiToken", apiToken)
				viper.Set("apiTokenEnvVar", nil)
			}

			if apiTokenEnvVar != "" {
				if err := confirmAuthReplacement("apiToken"); err != nil {
					return err
				}
				viper.Set("apiTokenEnvVar", apiTokenEnvVar)
				viper.Set("apiToken", nil)
			}

			viper.Set("serviceUrl", normalizedServiceURL)
			viper.Set("allowInsecureHTTP", allowInsecureHTTP)
			commandDebugf("writing config file=%s", viper.ConfigFileUsed())
			return viper.WriteConfig()
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			log.Print("config updated")
		},
	}
)

func init() {
	configCmd.Flags().String("apiToken", "", "API token")
	configCmd.Flags().String("apiTokenEnvVar", "", "Environment variable containing the API token")
	configCmd.Flags().String("serviceUrl", "https://dashboard.tensordock.com/api/v2", "Service URL")
	configCmd.Flags().Bool("allowInsecureHTTP", false, "Allow an insecure http service URL")
	configCmd.MarkFlagsMutuallyExclusive("apiToken", "apiTokenEnvVar")
	rootCmd.AddCommand(configCmd)
}

func confirmAuthReplacement(conflictingKey string) error {
	existingValue := viper.GetString(conflictingKey)
	if existingValue == "" {
		commandDebugf("config replacement check skipped key=%s", conflictingKey)
		return nil
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		commandDebugf("config replacement denied due to non-interactive stdin key=%s", conflictingKey)
		return fmt.Errorf("refusing to replace existing %s config without interactive confirmation", conflictingKey)
	}

	fmt.Printf("warning: existing %s configuration is set and will be replaced. Continue? [y/N]: ", conflictingKey)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes":
		commandDebugf("config replacement confirmed key=%s", conflictingKey)
		return nil
	default:
		commandDebugf("config replacement declined key=%s", conflictingKey)
		return fmt.Errorf("aborted without changing config")
	}
}
