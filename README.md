# tensordock-cli

A CLI client for the TensorDock v2 API.

## Installation

Install with `go install github.com/caguiclajmg/tensordock-cli` or build locally with:

```sh
go build
```

## Configuration

Store your TensorDock API token:

```sh
tensordock-cli config --apiToken your_token
```

Or store the environment variable name that should be read at runtime:

```sh
tensordock-cli config --apiTokenEnvVar TENSORDOCK_API_TOKEN
```

The default API base URL is `https://dashboard.tensordock.com/api/v2`.

You can also pass `--apiToken` or `--apiTokenEnvVar` on individual commands.

If you switch between stored auth types, the CLI warns and asks for confirmation before replacing the existing config entry.

## Supported Commands

### List instances

```sh
tensordock-cli servers list
```

### Get instance info

```sh
tensordock-cli servers info instance_id
```

### Start / stop / delete an instance

```sh
tensordock-cli servers start instance_id
tensordock-cli servers stop instance_id
tensordock-cli servers delete instance_id
```

### Create an instance

Location-based deployment:

```sh
tensordock-cli servers deploy my-instance \
  --locationId loc-uuid \
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
  --hostnodeId hostnode-uuid \
  --image ubuntu2404 \
  --gpuModel geforcertx4090-pcie-24gb \
  --gpuCount 1 \
  --vcpus 4 \
  --ram 8 \
  --storage 100 \
  --sshKeySecretId secret-uuid \
  --portForward 22:30022
```

Additional deploy options:

- `--dedicatedIp`
- `--cloudInitFile path/to/cloud-init.yaml`
- `--cloudInitRunCmd "echo hello"` (repeatable)
- `--cloudInitPackage curl` (repeatable)
- `--cloudInitPackageUpdate`
- `--cloudInitPackageUpgrade`
- `--cloudInitWriteFile 'path=/etc/motd,content=hello world[,owner=root:root][,permissions=0644]'` (repeatable)
- compatibility alias `--os` for simple image mapping

`--cloudInitFile` cannot be combined with the explicit `cloud_init` flags.

### Modify an instance

```sh
tensordock-cli servers modify instance_id \
  --cpuCores 8 \
  --ramGb 32 \
  --diskGb 200 \
  --gpuModel geforcertx4090-pcie-24gb \
  --gpuCount 1
```

Compatibility aliases remain available:

- `--vcpus` for `--cpuCores`
- `--ram` for `--ramGb`
- `--storage` for `--diskGb`

### Open an SSH session

```sh
tensordock-cli servers ssh instance_id
```

### Open an instance in the dashboard

```sh
tensordock-cli servers manage instance_id
```

`servers manage` opens the matching TensorDock dashboard page for that instance in your browser.

### Secrets

```sh
tensordock-cli secrets list
tensordock-cli secrets get secret_id
tensordock-cli secrets create --name my-key --type SSHKEY --value "ssh-ed25519 AAAA..."
tensordock-cli secrets delete secret_id
```

### Locations

```sh
tensordock-cli locations list
```

### Hostnodes

```sh
tensordock-cli hostnodes list --location loc-uuid --gpu geforcertx4090-pcie-24gb
tensordock-cli hostnodes info hostnode-uuid
```

## Removed Legacy Commands

These commands were removed because they depended on legacy endpoints that are not covered by the reviewed v2 API docs:

- `servers restart`
- `servers status`
- `billing`
- `stock list`
