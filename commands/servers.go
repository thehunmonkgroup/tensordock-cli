package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/caguiclajmg/tensordock-cli/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	serversCmd = &cobra.Command{
		Use:   "servers",
		Short: "Manage instances",
	}
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List instances",
		RunE:  serverList,
	}
	infoCmd = &cobra.Command{
		Use:   "info [flags] instance_id",
		Short: "Get instance info",
		Args:  cobra.ExactArgs(1),
		RunE:  serverInfo,
	}
	startCmd = &cobra.Command{
		Use:     "start [flags] instance_id",
		Short:   "Start an instance",
		Args:    cobra.ExactArgs(1),
		RunE:    startServer,
		PostRun: logAction("success"),
	}
	stopCmd = &cobra.Command{
		Use:     "stop [flags] instance_id",
		Short:   "Stop an instance",
		Args:    cobra.ExactArgs(1),
		RunE:    stopServer,
		PostRun: logAction("success"),
	}
	deleteCmd = &cobra.Command{
		Use:     "delete [flags] instance_id",
		Short:   "Delete an instance",
		Args:    cobra.ExactArgs(1),
		RunE:    deleteServer,
		PostRun: logAction("success"),
	}
	deployCmd = &cobra.Command{
		Use:     "deploy [flags] name [admin_user] [admin_pass]",
		Short:   "Create an instance",
		Args:    cobra.RangeArgs(1, 3),
		RunE:    deployServer,
		PostRun: logAction("success"),
	}
	manageCmd = &cobra.Command{
		Use:   "manage instance_id",
		Short: "Compatibility placeholder for dashboard management",
		Args:  cobra.ExactArgs(1),
		RunE:  manageServer,
	}
	sshCmd = &cobra.Command{
		Use:   "ssh instance_id",
		Short: "Launch an SSH session with an instance",
		Args:  cobra.ExactArgs(1),
		RunE:  sshServer,
	}
	modifyCmd = &cobra.Command{
		Use:     "modify [flags] instance_id",
		Short:   "Modify instance resources",
		Args:    cobra.ExactArgs(1),
		RunE:    modifyServer,
		PostRun: logAction("success"),
	}
)

func init() {
	serversCmd.AddCommand(listCmd)
	serversCmd.AddCommand(infoCmd)
	serversCmd.AddCommand(stopCmd)
	serversCmd.AddCommand(startCmd)
	serversCmd.AddCommand(deleteCmd)
	serversCmd.AddCommand(deployCmd)
	serversCmd.AddCommand(manageCmd)
	serversCmd.AddCommand(sshCmd)
	serversCmd.AddCommand(modifyCmd)

	deployCmd.Flags().String("locationId", "", "Location UUID for location-based deployment")
	deployCmd.Flags().String("hostnodeId", "", "Hostnode UUID for direct deployment")
	deployCmd.Flags().String("image", "ubuntu2404", "Image identifier")
	deployCmd.Flags().String("os", "", "Compatibility alias for --image")
	deployCmd.Flags().Bool("dedicatedIp", false, "Request a dedicated IP")
	deployCmd.Flags().StringArray("portForward", nil, "Port forward in internal:external form")
	deployCmd.Flags().String("sshKey", "", "SSH public key content")
	deployCmd.Flags().String("sshKeySecretId", "", "Secret ID containing an SSH key")
	deployCmd.Flags().String("cloudInitFile", "", "Path to JSON or YAML cloud-init object")
	deployCmd.Flags().String("gpuModel", "", "GPU model v0 name")
	deployCmd.Flags().Int("gpuCount", 0, "Number of GPUs")
	deployCmd.Flags().Int("vcpus", 4, "Number of vCPUs")
	deployCmd.Flags().Int("ram", 8, "RAM in GB")
	deployCmd.Flags().Int("storage", 100, "Storage in GB")

	sshCmd.Flags().String("bin", "ssh", "Name of SSH client executable")
	sshCmd.Flags().String("user", "user", "User account to use for login")
	sshCmd.Flags().String("extraFlags", "", "Extra flags to pass to the SSH client")

	modifyCmd.Flags().Int("cpuCores", 0, "CPU cores (step of 2)")
	modifyCmd.Flags().Int("ramGb", 0, "RAM in GB")
	modifyCmd.Flags().Int("diskGb", 0, "Disk in GB")
	modifyCmd.Flags().String("gpuModel", "", "GPU model v0 name")
	modifyCmd.Flags().Int("gpuCount", 0, "GPU count")
	modifyCmd.Flags().Int("vcpus", 0, "Compatibility alias for --cpuCores")
	modifyCmd.Flags().Int("ram", 0, "Compatibility alias for --ramGb")
	modifyCmd.Flags().Int("storage", 0, "Compatibility alias for --diskGb")

	rootCmd.AddCommand(serversCmd)
}

func serverList(cmd *cobra.Command, args []string) error {
	instances, err := client.ListInstances()
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Status"})
	for _, instance := range instances {
		name := instance.Name
		status := instance.Status
		if name == "" {
			name = instance.Attributes.Name
		}
		if status == "" {
			status = instance.Attributes.Status
		}
		t.AppendRow(table.Row{instance.ID, name, status})
	}
	t.Render()

	return nil
}

func serverInfo(cmd *cobra.Command, args []string) error {
	instance, err := client.GetInstance(args[0])
	if err != nil {
		return err
	}

	props := []table.Row{
		{"ID", instance.ID},
		{"Name", instance.Name},
		{"Status", instance.Status},
		{"IP Address", instance.IPAddress},
		{"Rate Hourly", fmt.Sprintf("%v", instance.RateHourly)},
		{"vCPUs", instance.Resources.VCPUCount},
		{"RAM (GB)", instance.Resources.RAMGB},
		{"Storage (GB)", instance.Resources.StorageGB},
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Property", "Value"})
	for _, row := range props {
		t.AppendRow(row)
	}

	if len(instance.Resources.GPUs) > 0 {
		for name, gpu := range instance.Resources.GPUs {
			t.AppendRow(table.Row{"GPU", fmt.Sprintf("%s x%d", firstNonEmpty(gpu.V0Name, name), gpu.Count)})
		}
	}
	if len(instance.PortForwards) > 0 {
		for _, portForward := range instance.PortForwards {
			t.AppendRow(table.Row{"Port Forward", fmt.Sprintf("%d:%d", portForward.InternalPort, portForward.ExternalPort)})
		}
	}
	t.Render()

	return nil
}

func startServer(cmd *cobra.Command, args []string) error {
	_, err := client.StartInstance(args[0])
	return err
}

func stopServer(cmd *cobra.Command, args []string) error {
	_, err := client.StopInstance(args[0])
	return err
}

func deleteServer(cmd *cobra.Command, args []string) error {
	_, err := client.DeleteInstance(args[0])
	return err
}

func deployServer(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	if len(args) > 1 {
		log.Print("warning: legacy admin_user/admin_pass arguments are ignored by the v2 API")
	}

	locationID, err := flags.GetString("locationId")
	if err != nil {
		return err
	}
	hostnodeID, err := flags.GetString("hostnodeId")
	if err != nil {
		return err
	}
	if locationID == "" && hostnodeID == "" {
		return errors.New("either --locationId or --hostnodeId is required")
	}
	if locationID != "" && hostnodeID != "" {
		return errors.New("--locationId and --hostnodeId are mutually exclusive")
	}

	image, err := resolvedImage(flags)
	if err != nil {
		return err
	}
	gpuModel, err := flags.GetString("gpuModel")
	if err != nil {
		return err
	}
	gpuCount, err := flags.GetInt("gpuCount")
	if err != nil {
		return err
	}
	vcpus, err := flags.GetInt("vcpus")
	if err != nil {
		return err
	}
	ram, err := flags.GetInt("ram")
	if err != nil {
		return err
	}
	storage, err := flags.GetInt("storage")
	if err != nil {
		return err
	}
	dedicatedIP, err := flags.GetBool("dedicatedIp")
	if err != nil {
		return err
	}
	portForwardFlags, err := flags.GetStringArray("portForward")
	if err != nil {
		return err
	}
	portForwards, err := parsePortForwards(portForwardFlags)
	if err != nil {
		return err
	}
	sshKey, err := resolveSSHKey(flags)
	if err != nil {
		return err
	}
	cloudInitRaw, err := readCloudInit(flags)
	if err != nil {
		return err
	}

	if storage < 100 {
		return errors.New("storage must be at least 100 GB")
	}
	if locationID != "" && gpuCount < 1 {
		return errors.New("location-based deployment requires at least one GPU")
	}
	if !strings.HasPrefix(strings.ToLower(image), "windows") && strings.TrimSpace(sshKey) == "" {
		return errors.New("an SSH key is required for non-Windows images")
	}

	request := api.InstanceCreateRequest{}
	request.Data.Type = "virtualmachine"
	request.Data.Attributes.Name = args[0]
	request.Data.Attributes.Type = "virtualmachine"
	request.Data.Attributes.Image = image
	request.Data.Attributes.Resources = api.InstanceResources{
		VCPUCount: vcpus,
		RAMGB:     ram,
		StorageGB: storage,
	}
	if gpuCount > 0 && gpuModel != "" {
		request.Data.Attributes.Resources.GPUs = map[string]api.GPUCount{
			gpuModel: {Count: gpuCount},
		}
	}
	request.Data.Attributes.LocationID = locationID
	request.Data.Attributes.HostnodeID = hostnodeID
	request.Data.Attributes.UseDedicatedIP = dedicatedIP
	request.Data.Attributes.PortForwards = portForwards
	request.Data.Attributes.SSHKey = sshKey
	request.Data.Attributes.CloudInit = cloudInitRaw

	instance, err := client.CreateInstance(request)
	if err != nil {
		return err
	}

	fmt.Println(instance.ID)
	return nil
}

func manageServer(cmd *cobra.Command, args []string) error {
	// TODO: Review whether the v2 API exposes dashboard management URLs for this command.
	_, err := client.GetInstance(args[0])
	if err != nil {
		return err
	}

	return errors.New("servers manage is retained for compatibility but no v2 dashboard URL is documented yet")
}

func sshServer(cmd *cobra.Command, args []string) error {
	instance, err := client.GetInstance(args[0])
	if err != nil {
		return err
	}
	if instance.IPAddress == "" {
		return errors.New("instance does not have a public IP address")
	}

	flags := cmd.Flags()
	bin, err := flags.GetString("bin")
	if err != nil {
		return err
	}
	user, err := flags.GetString("user")
	if err != nil {
		return err
	}
	extraFlags, err := flags.GetString("extraFlags")
	if err != nil {
		return err
	}

	argv := append(strings.Fields(extraFlags), fmt.Sprintf("%s@%s", user, instance.IPAddress))
	sshCmd := exec.Command(bin, argv...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func logAction(message string) func(*cobra.Command, []string) {
	return func(c *cobra.Command, s []string) { log.Println(message) }
}

func modifyServer(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	instance, err := client.GetInstance(args[0])
	if err != nil {
		return err
	}
	if instance.Status != "Stopped" && instance.Status != "StoppedDisassociated" && strings.ToLower(instance.Status) != "stopped" && strings.ToLower(instance.Status) != "stoppeddisassociated" {
		return errors.New("instance must be stopped or stopped-disassociated before modification")
	}

	cpuCores, err := intFlagAlias(flags, "cpuCores", "vcpus")
	if err != nil {
		return err
	}
	ramGB, err := intFlagAlias(flags, "ramGb", "ram")
	if err != nil {
		return err
	}
	diskGB, err := intFlagAlias(flags, "diskGb", "storage")
	if err != nil {
		return err
	}
	gpuModel, err := flags.GetString("gpuModel")
	if err != nil {
		return err
	}
	gpuCount, err := flags.GetInt("gpuCount")
	if err != nil {
		return err
	}

	if cpuCores == 0 && ramGB == 0 && diskGB == 0 && gpuModel == "" && gpuCount == 0 {
		return errors.New("at least one resource change is required")
	}
	if cpuCores != 0 && cpuCores%2 != 0 {
		return errors.New("cpuCores must be a multiple of 2")
	}
	if ramGB != 0 && !validModifyRAM(ramGB) {
		return errors.New("ramGb must be one of the documented allowed values")
	}
	if diskGB != 0 {
		if diskGB < 100 {
			return errors.New("diskGb must be at least 100")
		}
		if diskGB < instance.Resources.StorageGB {
			return errors.New("diskGb cannot decrease existing storage")
		}
	}

	request := api.InstanceModifyRequest{
		CPUCores: cpuCores,
		RAMGB:    ramGB,
		DiskGB:   diskGB,
	}
	if gpuModel != "" || gpuCount > 0 {
		if gpuModel == "" || gpuCount == 0 {
			return errors.New("both --gpuModel and --gpuCount are required when modifying GPUs")
		}
		request.GPUs = &struct {
			GPUV0Name string `json:"gpuV0Name"`
			Count     int    `json:"count"`
		}{
			GPUV0Name: gpuModel,
			Count:     gpuCount,
		}
	}

	_, err = client.ModifyInstance(args[0], request)
	return err
}

func parsePortForwards(values []string) ([]api.PortForward, error) {
	portForwards := make([]api.PortForward, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port forward %q: expected internal:external", value)
		}
		internalPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid internal port %q", parts[0])
		}
		externalPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid external port %q", parts[1])
		}
		portForwards = append(portForwards, api.PortForward{
			InternalPort: internalPort,
			ExternalPort: externalPort,
		})
	}
	return portForwards, nil
}

func resolvedImage(flags *pflag.FlagSet) (string, error) {
	image, err := flags.GetString("image")
	if err != nil {
		return "", err
	}
	osValue, err := flags.GetString("os")
	if err != nil {
		return "", err
	}
	if osValue == "" {
		return image, nil
	}

	switch strings.ToLower(osValue) {
	case "ubuntu 24.04", "ubuntu 24.04 lts", "ubuntu2404":
		return "ubuntu2404", nil
	case "windows 10", "windows10":
		return "windows10", nil
	default:
		return "", fmt.Errorf("unsupported --os value %q; use --image for explicit control", osValue)
	}
}

func resolveSSHKey(flags *pflag.FlagSet) (string, error) {
	sshKey, err := flags.GetString("sshKey")
	if err != nil {
		return "", err
	}
	secretID, err := flags.GetString("sshKeySecretId")
	if err != nil {
		return "", err
	}
	if sshKey != "" && secretID != "" {
		return "", errors.New("--sshKey and --sshKeySecretId are mutually exclusive")
	}
	if secretID == "" {
		return sshKey, nil
	}

	secret, err := client.GetSecret(secretID)
	if err != nil {
		return "", err
	}
	if secret.Value == "" {
		return "", errors.New("the selected secret did not return a usable SSH key value")
	}
	return secret.Value, nil
}

func readCloudInit(flags *pflag.FlagSet) (json.RawMessage, error) {
	path, err := flags.GetString("cloudInitFile")
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		if err := yaml.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("cloudInitFile must contain valid JSON or YAML")
		}
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(normalized), nil
}

func intFlagAlias(flags *pflag.FlagSet, primary string, alias string) (int, error) {
	if flags.Changed(primary) {
		return flags.GetInt(primary)
	}
	if flags.Changed(alias) {
		return flags.GetInt(alias)
	}
	return 0, nil
}

func validModifyRAM(value int) bool {
	allowed := map[int]bool{
		2: true, 4: true, 6: true, 8: true, 10: true, 16: true, 32: true, 48: true,
		64: true, 80: true, 96: true, 112: true, 128: true, 144: true, 160: true,
		176: true, 192: true, 208: true, 224: true, 240: true, 256: true, 512: true,
	}
	return allowed[value]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
