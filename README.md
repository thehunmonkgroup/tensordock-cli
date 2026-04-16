# tensordock-cli

A CLI client for the [TensorDock v2 API](https://dashboard.tensordock.com/api/docs).

## Installation

Install with:

```sh
go install github.com/thehunmonkgroup/tensordock-cli@latest
```

Build locally with Go 1.23+:

```sh
go build
```

Or grab one of the binaries from an official release.

## Quick Start

If `TENSORDOCK_API_TOKEN` is set, the CLI uses it automatically by default:

```sh
export TENSORDOCK_API_TOKEN=your_token
tensordock-cli servers list
```

To store configuration instead, use one of:

```sh
tensordock-cli config --apiToken your_token
# Set a custom variable name to read the API token from 
tensordock-cli config --apiTokenEnvVar CUSTOM_API_TOKEN
```

You can also pass `--apiToken` or `--apiTokenEnvVar` on individual commands.

To store a default SSH login user for `servers ssh`:

```sh
tensordock-cli config --sshUser ubuntu
```

## Configuration Notes

By default, config is stored in `config.yml` under the OS user config directory for `tensordock-cli`. On Linux this is typically `$XDG_CONFIG_HOME/tensordock-cli/config.yml`, or `~/.config/tensordock-cli/config.yml` when `XDG_CONFIG_HOME` is not set.

Use `--configDir` to override the config directory location.

Deploy templates live under the OS user config directory in `templates/`. On Linux this is typically `$XDG_CONFIG_HOME/tensordock-cli/templates`, or `~/.config/tensordock-cli/templates` when `XDG_CONFIG_HOME` is not set.

Use `--templateDir` to override the template directory location.

The default API base URL is `https://dashboard.tensordock.com/api/v2`.

To use a different API endpoint:

```sh
tensordock-cli config --serviceUrl https://example.com/api/v2
```

`tensordock-cli config` writes only the values you explicitly pass, so the config file stays minimal instead of being populated with default values.

Custom `http://` service URLs are rejected unless you explicitly opt in with `--allowInsecureHTTP` or store that setting with `tensordock-cli config`.

If you switch between stored auth types, the CLI warns and asks for confirmation before replacing the existing config entry.

## Common Commands

```sh
tensordock-cli servers list
tensordock-cli servers info [instance_id_or_name]
tensordock-cli servers start [instance_id_or_name]
tensordock-cli servers stop [instance_id_or_name]
tensordock-cli servers delete [instance_id_or_name]
tensordock-cli servers manage [instance_id_or_name]
tensordock-cli secrets list
tensordock-cli locations list --gpu geforcertx4090-pcie-24gb
tensordock-cli hostnodes list --location [location_id] --gpu geforcertx4090-pcie-24gb
```

Use `tensordock-cli --help` or `tensordock-cli <command> --help` for the full command surface.

## Deploy an Instance

Location-based deployment:

```sh
tensordock-cli servers deploy my-instance \
  --locationId [location_id] \
  --image ubuntu2404 \
  --gpuModel geforcertx4090-pcie-24gb \
  --gpuCount 1 \
  --vcpus 4 \
  --ram 8 \
  --storage 100 \
  --sshKey "$(cat ~/.ssh/id_ed25519.pub)"
```

Hostnode-based deployment:

```sh
tensordock-cli servers deploy my-instance \
  --hostnodeId [hostnode_id] \
  --image ubuntu2404 \
  --gpuModel geforcertx4090-pcie-24gb \
  --gpuCount 1 \
  --vcpus 4 \
  --ram 8 \
  --storage 100 \
  --sshKeySecretId [secret_id] \
  --portForward 22:30022
```

Additional deploy options:

- `--dedicatedIp`
- `--template name` to load `name.yml` from the template directory
- `--cloudInitFile path/to/cloud-init.yaml`
- `--cloudInitRunCmd "echo hello"` (repeatable)
- `--cloudInitPackage curl` (repeatable)
- `--cloudInitPackageUpdate`
- `--cloudInitPackageUpgrade`
- `--cloudInitWriteFile 'path=/etc/motd,content=hello world[,owner=root:root][,permissions=0644]'` (repeatable)
- compatibility alias `--os` for simple image mapping

`--cloudInitFile` cannot be combined with the explicit `cloud_init` flags.

Deploy template example:

```yaml
locationId: [location_id]
image: ubuntu2404
gpuModel: geforcertx4090-pcie-24gb
gpuCount: 1
vcpus: 4
ram: 8
storage: 100
sshKeySecretId: [secret_id]
portForward:
  - "22:30022"
cloudInitRunCmd:
  - echo hello
```

If the file is stored as `~/.config/tensordock-cli/templates/foo.yml`, deploy with:

```sh
tensordock-cli servers deploy my-instance --template foo
```

Explicit deploy flags override values loaded from the template:

```sh
tensordock-cli servers deploy my-instance --template foo --ram 16
```

## Modify or Access an Instance

```sh
tensordock-cli servers modify [instance_id_or_name] \
  --cpuCores 8 \
  --ramGb 32 \
  --diskGb 200 \
  --gpuModel geforcertx4090-pcie-24gb \
  --gpuCount 1
```

Compatibility aliases:

- `--vcpus` for `--cpuCores`
- `--ram` for `--ramGb`
- `--storage` for `--diskGb`

```sh
tensordock-cli servers ssh [instance_id_or_name]
tensordock-cli servers ssh [instance_id_or_name] --sshUser ubuntu
tensordock-cli servers ssh [instance_id_or_name] --extraFlags="-i /path/to/key -p 2222"
```
