package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thehunmonkgroup/tensordock-cli/api"
	"gopkg.in/yaml.v3"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Set API token configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			apiTokenChanged := flags.Changed("apiToken")
			apiTokenEnvVarChanged := flags.Changed("apiTokenEnvVar")
			serviceURLChanged := flags.Changed("serviceUrl")
			allowInsecureHTTPChanged := flags.Changed("allowInsecureHTTP")

			if !apiTokenChanged && !apiTokenEnvVarChanged && !serviceURLChanged && !allowInsecureHTTPChanged {
				return fmt.Errorf("at least one config flag must be provided")
			}

			apiToken, err := flags.GetString("apiToken")
			if err != nil {
				return err
			}
			if apiTokenChanged && apiToken == "" {
				return fmt.Errorf("api token cannot be empty")
			}

			apiTokenEnvVar, err := flags.GetString("apiTokenEnvVar")
			if err != nil {
				return err
			}
			if apiTokenEnvVarChanged && apiTokenEnvVar == "" {
				return fmt.Errorf("api token environment variable name cannot be empty")
			}

			updates := make(map[string]any)

			if apiTokenChanged {
				if err := confirmAuthReplacement("apiTokenEnvVar"); err != nil {
					return err
				}
				updates["apiToken"] = apiToken
				updates["apiTokenEnvVar"] = nil
			}

			if apiTokenEnvVarChanged {
				if err := confirmAuthReplacement("apiToken"); err != nil {
					return err
				}
				updates["apiTokenEnvVar"] = apiTokenEnvVar
				updates["apiToken"] = nil
			}

			if serviceURLChanged {
				serviceURL, err := flags.GetString("serviceUrl")
				if err != nil {
					return err
				}

				allowInsecureHTTP := viper.GetBool("allowInsecureHTTP")
				if allowInsecureHTTPChanged {
					allowInsecureHTTP, err = flags.GetBool("allowInsecureHTTP")
					if err != nil {
						return err
					}
				}

				normalizedServiceURL, err := api.ValidateBaseURL(serviceURL, allowInsecureHTTP)
				if err != nil {
					return err
				}
				updates["serviceUrl"] = normalizedServiceURL
			}

			if allowInsecureHTTPChanged {
				allowInsecureHTTP, err := flags.GetBool("allowInsecureHTTP")
				if err != nil {
					return err
				}

				serviceURL := viper.GetString("serviceUrl")
				if serviceURLChanged {
					serviceURL, err = flags.GetString("serviceUrl")
					if err != nil {
						return err
					}
				}

				normalizedServiceURL, err := api.ValidateBaseURL(serviceURL, allowInsecureHTTP)
				if err != nil {
					return err
				}
				updates["allowInsecureHTTP"] = allowInsecureHTTP
				if serviceURLChanged {
					updates["serviceUrl"] = normalizedServiceURL
				}
			}

			commandDebugf(
				"config update requested api_token_changed=%t api_token_env_var_changed=%t service_url_changed=%t allow_insecure_http_changed=%t",
				apiTokenChanged,
				apiTokenEnvVarChanged,
				serviceURLChanged,
				allowInsecureHTTPChanged,
			)
			commandDebugf("writing config file=%s", viper.ConfigFileUsed())
			return writeExplicitConfigUpdates(viper.ConfigFileUsed(), updates)
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
	if !viper.InConfig(conflictingKey) {
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

func writeExplicitConfigUpdates(configPath string, updates map[string]any) error {
	configValues, err := loadConfigValues(configPath)
	if err != nil {
		return err
	}

	for key, value := range updates {
		if value == nil {
			deleteConfigKey(configValues, key)
			continue
		}
		setConfigKey(configValues, key, value)
	}

	return writeConfigValues(configPath, configValues)
}

func loadConfigValues(configPath string) (map[string]any, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return make(map[string]any), nil
	}

	values := make(map[string]any)
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}

	if values == nil {
		values = make(map[string]any)
	}

	return values, nil
}

func writeConfigValues(configPath string, values map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o600)
}

func setConfigKey(values map[string]any, key string, value any) {
	if existingKey, ok := findConfigKey(values, key); ok {
		values[existingKey] = value
		return
	}

	values[strings.ToLower(key)] = value
}

func deleteConfigKey(values map[string]any, key string) {
	if existingKey, ok := findConfigKey(values, key); ok {
		delete(values, existingKey)
	}
}

func findConfigKey(values map[string]any, key string) (string, bool) {
	for existingKey := range values {
		if strings.EqualFold(existingKey, key) {
			return existingKey, true
		}
	}

	return "", false
}
