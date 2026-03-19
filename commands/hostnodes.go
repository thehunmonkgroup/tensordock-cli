package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/caguiclajmg/tensordock-cli/debugutil"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	hostnodesCmd = &cobra.Command{
		Use:   "hostnodes",
		Short: "Inspect hostnodes",
	}
	hostnodesListCmd = &cobra.Command{
		Use:   "list",
		Short: "List hostnodes",
		RunE:  listHostnodes,
	}
	hostnodesInfoCmd = &cobra.Command{
		Use:   "info hostnode_id",
		Short: "Get hostnode details",
		Args:  cobra.ExactArgs(1),
		RunE:  getHostnode,
	}
)

func init() {
	hostnodesCmd.AddCommand(hostnodesListCmd)
	hostnodesCmd.AddCommand(hostnodesInfoCmd)

	hostnodesListCmd.Flags().String("location", "", "Location UUID filter")
	hostnodesListCmd.Flags().String("minRamGb", "", "Minimum RAM in GB")
	hostnodesListCmd.Flags().String("minVcpu", "", "Minimum vCPU count")
	hostnodesListCmd.Flags().String("gpu", "", "GPU v0 name filter")

	rootCmd.AddCommand(hostnodesCmd)
}

func listHostnodes(cmd *cobra.Command, args []string) error {
	filters := map[string]string{}
	for _, key := range []string{"location", "minRamGb", "minVcpu", "gpu"} {
		value, err := cmd.Flags().GetString(key)
		if err != nil {
			return err
		}
		if value != "" {
			filters[key] = value
		}
	}
	commandDebugf("listing hostnodes filters=%s", debugutil.FormatStringMap(filters))

	hostnodes, err := client.ListHostnodes(filters)
	if err != nil {
		return err
	}
	commandDebugf("listing hostnodes result_count=%d", len(hostnodes))

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Location", "Engine", "vCPUs", "RAM", "Storage", "GPUs"})
	for _, hostnode := range hostnodes {
		gpus := make([]string, 0, len(hostnode.AvailableResources.GPUs))
		for _, gpu := range hostnode.AvailableResources.GPUs {
			gpus = append(gpus, fmt.Sprintf("%s x%d", gpu.V0Name, gpu.AvailableCount))
		}
		location := strings.TrimSpace(strings.Join([]string{hostnode.Location.City, hostnode.Location.StateProvince, hostnode.Location.Country}, ", "))
		t.AppendRow(table.Row{
			hostnode.ID,
			location,
			hostnode.Engine,
			hostnode.AvailableResources.VCPUCount,
			hostnode.AvailableResources.RAMGB,
			hostnode.AvailableResources.StorageGB,
			strings.Join(gpus, "; "),
		})
	}
	t.Render()

	return nil
}

func getHostnode(cmd *cobra.Command, args []string) error {
	commandDebugf("fetching hostnode id=%s", args[0])
	hostnode, err := client.GetHostnode(args[0])
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Property", "Value"})
	t.AppendRows([]table.Row{
		{"ID", hostnode.ID},
		{"Engine", hostnode.Engine},
		{"Location", strings.TrimSpace(strings.Join([]string{hostnode.Location.City, hostnode.Location.StateProvince, hostnode.Location.Country}, ", "))},
		{"Tier", hostnode.Location.Tier},
		{"Max vCPUs", hostnode.AvailableResources.MaxVCPUs},
		{"Max RAM (GB)", hostnode.AvailableResources.MaxRAMGB},
		{"Max Storage (GB)", hostnode.AvailableResources.MaxStorageGB},
		{"Public IP Available", hostnode.AvailableResources.HasPublicIPAvailable},
		{"Available Ports", fmt.Sprintf("%v", hostnode.AvailableResources.AvailablePorts)},
	})
	for _, gpu := range hostnode.AvailableResources.GPUs {
		t.AppendRow(table.Row{"GPU", fmt.Sprintf("%s x%d", gpu.V0Name, gpu.AvailableCount)})
	}
	t.Render()

	return nil
}
