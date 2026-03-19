package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	locationsCmd = &cobra.Command{
		Use:   "locations",
		Short: "List deployment locations",
	}
	locationsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List locations",
		RunE:  listLocations,
	}
)

func init() {
	locationsCmd.AddCommand(locationsListCmd)
	rootCmd.AddCommand(locationsCmd)
}

func listLocations(cmd *cobra.Command, args []string) error {
	commandDebugf("listing locations")
	locations, err := client.ListLocations()
	if err != nil {
		return err
	}
	commandDebugf("listing locations result_count=%d", len(locations))

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Location", "Tier", "GPU Offerings"})
	for _, location := range locations {
		gpuNames := make([]string, 0, len(location.GPUs))
		for _, gpu := range location.GPUs {
			gpuNames = append(gpuNames, fmt.Sprintf("%s x%d", gpu.V0Name, gpu.MaxCount))
		}
		label := strings.TrimSpace(strings.Join([]string{location.City, location.StateProvince, location.Country}, ", "))
		t.AppendRow(table.Row{location.ID, label, location.Tier, strings.Join(gpuNames, "; ")})
	}
	t.Render()

	return nil
}
