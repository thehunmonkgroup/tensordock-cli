package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/thehunmonkgroup/tensordock-cli/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	secretsCmd = &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets",
	}
	secretsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List secrets",
		RunE:  listSecrets,
	}
	secretsGetCmd = &cobra.Command{
		Use:   "get secret_id",
		Short: "Get secret details",
		Args:  cobra.ExactArgs(1),
		RunE:  getSecret,
	}
	secretsCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a secret",
		RunE:  createSecret,
	}
	secretsDeleteCmd = &cobra.Command{
		Use:   "delete secret_id",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteSecret,
	}
)

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsCreateCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)

	secretsCreateCmd.Flags().String("name", "", "Secret name")
	secretsCreateCmd.Flags().String("type", "", "Secret type")
	secretsCreateCmd.Flags().String("value", "", "Secret value")
	secretsCreateCmd.MarkFlagRequired("name")
	secretsCreateCmd.MarkFlagRequired("type")
	secretsCreateCmd.MarkFlagRequired("value")

	rootCmd.AddCommand(secretsCmd)
}

func listSecrets(cmd *cobra.Command, args []string) error {
	commandDebugf("listing secrets")
	secrets, err := client.ListSecrets(cmd.Context())
	if err != nil {
		return err
	}
	commandDebugf("listing secrets result_count=%d", len(secrets))

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Type"})
	for _, secret := range secrets {
		t.AppendRow(table.Row{secret.ID, secret.Name, secret.Type})
	}
	t.Render()

	return nil
}

func getSecret(cmd *cobra.Command, args []string) error {
	commandDebugf("fetching secret id=%s", args[0])
	secret, err := client.GetSecret(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	fmt.Printf("ID: %s\nName: %s\nType: %s\n", secret.ID, secret.Name, secret.Type)
	if secret.Value != "" {
		fmt.Printf("Value: %s\n", secret.Value)
	}
	return nil
}

func createSecret(cmd *cobra.Command, args []string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	secretType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	value, err := cmd.Flags().GetString("value")
	if err != nil {
		return err
	}
	if secretType == "" {
		return errors.New("secret type is required")
	}
	commandDebugf("creating secret name=%q type=%q value_set=%t", name, secretType, value != "")

	request := api.SecretCreateRequest{}
	request.Data.Type = "secret"
	request.Data.Attributes = api.SecretCreateAttributes{
		Name:  name,
		Value: value,
		Type:  secretType,
	}

	secret, err := client.CreateSecret(cmd.Context(), request)
	if err != nil {
		return err
	}

	fmt.Println(secret.ID)
	return nil
}

func deleteSecret(cmd *cobra.Command, args []string) error {
	commandDebugf("deleting secret id=%s", args[0])
	_, err := client.DeleteSecret(cmd.Context(), args[0])
	return err
}
