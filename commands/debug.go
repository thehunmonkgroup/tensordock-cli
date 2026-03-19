package commands

import (
	"fmt"

	"github.com/caguiclajmg/tensordock-cli/debugutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func debugEnabled() bool {
	return viper.GetBool("debug")
}

func commandDebugf(format string, args ...interface{}) {
	debugutil.Logf(debugEnabled(), format, args...)
}

func describeAPITokenSource(cmd *cobra.Command) (string, error) {
	if cmd != nil {
		if cmd.Flags().Changed("apiToken") {
			return "flag:apiToken", nil
		}
		if cmd.Flags().Changed("apiTokenEnvVar") {
			name, err := cmd.Flags().GetString("apiTokenEnvVar")
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("flag:apiTokenEnvVar(%s)", name), nil
		}
	}

	if viper.GetString("apiToken") != "" {
		return "config:apiToken", nil
	}

	if name := viper.GetString("apiTokenEnvVar"); name != "" {
		return fmt.Sprintf("config:apiTokenEnvVar(%s)", name), nil
	}

	return "unconfigured", nil
}

func changedFlagName(flags *pflag.FlagSet, primary string, alias string) string {
	if flags.Changed(primary) {
		return primary
	}
	if flags.Changed(alias) {
		return alias
	}
	return ""
}
