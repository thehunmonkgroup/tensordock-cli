package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/thehunmonkgroup/tensordock-cli/api"
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
	locationsListCmd.Flags().String("gpu", "", "GPU v0 name filter")
	rootCmd.AddCommand(locationsCmd)
}

func listLocations(cmd *cobra.Command, args []string) error {
	commandDebugf("listing locations")
	locations, err := client.ListLocations(cmd.Context())
	if err != nil {
		return err
	}
	gpuFilter, err := cmd.Flags().GetString("gpu")
	if err != nil {
		return err
	}
	locations = filterLocationsByGPU(locations, gpuFilter)
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

func filterLocationsByGPU(locations []api.Location, gpuFilter string) []api.Location {
	gpuFilter = strings.TrimSpace(gpuFilter)
	if gpuFilter == "" {
		return locations
	}

	filtered := make([]api.Location, 0, len(locations))
	for _, location := range locations {
		if locationHasGPU(location, gpuFilter) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

func locationHasGPU(location api.Location, gpuFilter string) bool {
	for _, gpu := range location.GPUs {
		if strings.EqualFold(gpu.V0Name, gpuFilter) {
			return true
		}
	}

	return false
}
