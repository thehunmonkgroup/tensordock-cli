package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/caguiclajmg/tensordock-cli/api"
	"github.com/caguiclajmg/tensordock-cli/debugutil"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
		Use:     "deploy [flags] name",
		Short:   "Create an instance",
		Args:    cobra.ExactArgs(1),
		RunE:    deployServer,
		PostRun: logAction("success"),
	}
	manageCmd = &cobra.Command{
		Use:   "manage instance_id",
		Short: "Open an instance in the TensorDock dashboard",
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
	deployCmd.Flags().StringArray("cloudInitRunCmd", nil, "cloud_init runcmd entry; repeat flag to add multiple commands")
	deployCmd.Flags().StringArray("cloudInitPackage", nil, "cloud_init package entry; repeat flag to add multiple packages")
	deployCmd.Flags().Bool("cloudInitPackageUpdate", false, "Set cloud_init package_update")
	deployCmd.Flags().Bool("cloudInitPackageUpgrade", false, "Set cloud_init package_upgrade")
	deployCmd.Flags().StringArray("cloudInitWriteFile", nil, "cloud_init write_files entry as path=<...>,content=<...>[,owner=<...>][,permissions=<...>]")
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
	commandDebugf("listing servers")
	instances, err := client.ListInstances()
	if err != nil {
		return err
	}
	commandDebugf("listing servers result_count=%d", len(instances))

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
	commandDebugf("fetching server info id=%s", args[0])
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
	commandDebugf("starting server id=%s", args[0])
	_, err := client.StartInstance(args[0])
	return err
}

func stopServer(cmd *cobra.Command, args []string) error {
	commandDebugf("stopping server id=%s", args[0])
	_, err := client.StopInstance(args[0])
	return err
}

func deleteServer(cmd *cobra.Command, args []string) error {
	commandDebugf("deleting server id=%s", args[0])
	_, err := client.DeleteInstance(args[0])
	return err
}

func deployServer(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
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
	commandDebugf("deploy target name=%q location_id=%q hostnode_id=%q", args[0], locationID, hostnodeID)

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
	sshKey, sshKeySource, err := resolveSSHKey(flags)
	if err != nil {
		return err
	}
	cloudInitRaw, cloudInitFormat, err := resolveCloudInit(flags)
	if err != nil {
		return err
	}

	if storage < 100 {
		return errors.New("storage must be at least 100 GB")
	}
	if (gpuModel == "") != (gpuCount == 0) {
		return errors.New("both --gpuModel and --gpuCount are required when specifying GPUs")
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

	deploySummary := map[string]interface{}{
		"name":             args[0],
		"image":            image,
		"location_id":      locationID,
		"hostnode_id":      hostnodeID,
		"gpu_model":        gpuModel,
		"gpu_count":        gpuCount,
		"vcpus":            vcpus,
		"ram_gb":           ram,
		"storage_gb":       storage,
		"dedicated_ip":     dedicatedIP,
		"port_forwards":    len(portForwards),
		"ssh_key_source":   sshKeySource,
		"ssh_key":          sshKey,
		"cloud_init_type":  cloudInitFormat,
		"cloud_init_bytes": len(cloudInitRaw),
		"cloud_init":       cloudInitRaw,
	}
	commandDebugf("deploy request summary=%s", debugJSONSummary(deploySummary))

	instance, err := client.CreateInstance(request)
	if err != nil {
		return err
	}

	fmt.Println(instance.ID)
	return nil
}

func manageServer(cmd *cobra.Command, args []string) error {
	commandDebugf("preparing dashboard management URL for server id=%s", args[0])
	_, err := client.GetInstance(args[0])
	if err != nil {
		return err
	}

	dashboardURL, err := buildInstanceDashboardURL(args[0])
	if err != nil {
		return err
	}

	commandDebugf("launching dashboard URL for server id=%s url=%s", args[0], debugutil.RedactURL(dashboardURL))
	return openBrowser(dashboardURL)
}

func sshServer(cmd *cobra.Command, args []string) error {
	commandDebugf("preparing ssh session for server id=%s", args[0])
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
	commandDebugf("launching ssh bin=%q user=%q destination=%q extra_arg_count=%d", bin, user, instance.IPAddress, len(argv)-1)
	sshCmd := exec.Command(bin, argv...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func buildInstanceDashboardURL(instanceID string) (string, error) {
	serviceURL := strings.TrimSpace(viper.GetString("serviceUrl"))
	if serviceURL == "" {
		return "", errors.New("service URL is not configured")
	}

	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("invalid service URL %q: %w", serviceURL, err)
	}

	basePath := strings.TrimSuffix(strings.TrimRight(baseURL.Path, "/"), "/api/v2")
	if basePath == "" {
		basePath = "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	baseURL.Path = path.Join(basePath, "my-servers", instanceID)
	baseURL.RawPath = ""
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL.String(), nil
}

func openBrowser(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
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
	commandDebugf("modifying server id=%s current_status=%q", args[0], instance.Status)
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
	commandDebugf(
		"modify resource request id=%s cpu=%d cpu_source=%q ram=%d ram_source=%q disk=%d disk_source=%q gpu_model=%q gpu_count=%d",
		args[0],
		cpuCores,
		changedFlagName(flags, "cpuCores", "vcpus"),
		ramGB,
		changedFlagName(flags, "ramGb", "ram"),
		diskGB,
		changedFlagName(flags, "diskGb", "storage"),
		gpuModel,
		gpuCount,
	)

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

func resolveSSHKey(flags *pflag.FlagSet) (string, string, error) {
	sshKey, err := flags.GetString("sshKey")
	if err != nil {
		return "", "", err
	}
	secretID, err := flags.GetString("sshKeySecretId")
	if err != nil {
		return "", "", err
	}
	if sshKey != "" && secretID != "" {
		return "", "", errors.New("--sshKey and --sshKeySecretId are mutually exclusive")
	}
	if secretID == "" {
		if strings.TrimSpace(sshKey) == "" {
			return sshKey, "none", nil
		}
		return sshKey, "inline", nil
	}

	secret, err := client.GetSecret(secretID)
	if err != nil {
		return "", "", err
	}
	if secret.Value == "" {
		return "", "", errors.New("the selected secret did not return a usable SSH key value")
	}
	commandDebugf("resolved ssh key from secret id=%s", secretID)
	return secret.Value, fmt.Sprintf("secret:%s", secretID), nil
}

func resolveCloudInit(flags *pflag.FlagSet) (json.RawMessage, string, error) {
	filePath, err := flags.GetString("cloudInitFile")
	if err != nil {
		return nil, "", err
	}

	explicitRaw, explicitFormat, err := buildCloudInitFromFlags(flags)
	if err != nil {
		return nil, "", err
	}
	if filePath != "" && explicitRaw != nil {
		return nil, "", errors.New("--cloudInitFile cannot be combined with other cloud-init flags")
	}
	if filePath != "" {
		return readCloudInitFile(filePath)
	}
	if explicitRaw != nil {
		return explicitRaw, explicitFormat, nil
	}

	return nil, "none", nil
}

func readCloudInitFile(filePath string) (json.RawMessage, string, error) {
	if filePath == "" {
		return nil, "none", nil
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	var value interface{}
	format := "json"
	if err := json.Unmarshal(raw, &value); err != nil {
		if err := yaml.Unmarshal(raw, &value); err != nil {
			return nil, "", errors.New("cloudInitFile must contain valid JSON or YAML")
		}
		format = "yaml"
	}
	commandDebugf("loaded cloud-init file=%s format=%s bytes=%d", filePath, format, len(raw))

	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	return json.RawMessage(normalized), format, nil
}

func buildCloudInitFromFlags(flags *pflag.FlagSet) (json.RawMessage, string, error) {
	runCmds, err := flags.GetStringArray("cloudInitRunCmd")
	if err != nil {
		return nil, "", err
	}
	packages, err := flags.GetStringArray("cloudInitPackage")
	if err != nil {
		return nil, "", err
	}
	packageUpdate, err := flags.GetBool("cloudInitPackageUpdate")
	if err != nil {
		return nil, "", err
	}
	packageUpgrade, err := flags.GetBool("cloudInitPackageUpgrade")
	if err != nil {
		return nil, "", err
	}
	writeFileFlags, err := flags.GetStringArray("cloudInitWriteFile")
	if err != nil {
		return nil, "", err
	}

	if len(runCmds) == 0 && len(packages) == 0 && len(writeFileFlags) == 0 &&
		!flags.Changed("cloudInitPackageUpdate") && !flags.Changed("cloudInitPackageUpgrade") {
		return nil, "", nil
	}

	cloudInit := map[string]interface{}{}
	if len(runCmds) > 0 {
		cloudInit["runcmd"] = runCmds
	}
	if len(packages) > 0 {
		cloudInit["packages"] = packages
	}
	if flags.Changed("cloudInitPackageUpdate") {
		cloudInit["package_update"] = packageUpdate
	}
	if flags.Changed("cloudInitPackageUpgrade") {
		cloudInit["package_upgrade"] = packageUpgrade
	}
	if len(writeFileFlags) > 0 {
		writeFiles := make([]map[string]string, 0, len(writeFileFlags))
		for _, value := range writeFileFlags {
			writeFile, err := parseCloudInitWriteFile(value)
			if err != nil {
				return nil, "", err
			}
			writeFiles = append(writeFiles, writeFile)
		}
		cloudInit["write_files"] = writeFiles
	}

	raw, err := json.Marshal(cloudInit)
	if err != nil {
		return nil, "", err
	}
	return json.RawMessage(raw), "flags", nil
}

func parseCloudInitWriteFile(value string) (map[string]string, error) {
	writeFile := map[string]string{}
	for _, field := range strings.Split(value, ",") {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cloud-init write_files entry %q: expected key=value segments", value)
		}

		key := strings.TrimSpace(parts[0])
		fieldValue := parts[1]
		switch key {
		case "path", "content", "owner", "permissions":
			if strings.TrimSpace(fieldValue) == "" {
				return nil, fmt.Errorf("invalid cloud-init write_files entry %q: %s cannot be empty", value, key)
			}
			writeFile[key] = fieldValue
		default:
			return nil, fmt.Errorf("invalid cloud-init write_files entry %q: unsupported key %q", value, key)
		}
	}

	if writeFile["path"] == "" || writeFile["content"] == "" {
		return nil, fmt.Errorf("invalid cloud-init write_files entry %q: path and content are required", value)
	}

	return writeFile, nil
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

func debugJSONSummary(value interface{}) string {
	raw, err := json.Marshal(debugutil.Sanitize(value))
	if err != nil {
		return "<unavailable>"
	}

	return string(raw)
}
