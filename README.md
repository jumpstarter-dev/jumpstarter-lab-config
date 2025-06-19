# jumpstarter-lab-config

This repository proposes a gitops/configuration system and a tool for managing labs
and environments with jumpstarter controllers, exporter hosts (sidekicks), exporters,
and clients in enterprise environments.

## Installation

### Quick Install (Recommended)

Install the latest version using the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/jumpstarter-dev/jumpstarter-lab-config/main/install.sh | bash
```

Or install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/jumpstarter-dev/jumpstarter-lab-config/main/install.sh | bash -s v0.0.2
```

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/jumpstarter-dev/jumpstarter-lab-config/releases)
2. Extract the binary for your platform
3. Move it to a directory in your PATH (e.g., `~/.local/bin` or `/usr/local/bin`)
4. Make it executable: `chmod +x jumpstarter-lab-config`

### Verify Installation

After installation, verify it works:

```bash
jumpstarter-lab-config --help
```

Make sure `~/.local/bin` is in your PATH by adding this to your shell profile:

```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

## Features

* Linting of configuration files, to ensure they are valid and follow the expected format.
* Dry-run mode to verify changes before applying them.
* Apply mode to apply changes to the jumpstarter controllers.
* Support for multiple configuration files, allowing for modular and extensible configurations.
* Bases exporter host management on bootc.

## Usage

### Linting changes, useful to verify the configuration files in merge requests
```shell
$ jumpstarter-lab-config lint

Validating files from:
- clients/*.yaml
- devices/**/*.yaml
- exporter-templates/**/*.yaml
- locations/*.yaml
- jumpstarter-controllers/*.yaml

✅ All configurations are valid

```

You can also specify a custom configuration file:
```shell
$ jumpstarter-lab-config lint my-lab-config.yaml

Validating files from my-lab-config.yaml...

✅ All configurations are valid
```

Something is wrong with the configuration? The `lint` command will help you to find it.

```shell
$ jumpstarter-lab-config lint

Validating files from:
- clients/*.yaml
- devices/**/*.yaml
- exporter-templates/**/*.yaml
- locations/*.yaml
- jumpstarter-controllers/*.yaml

❌ Error in clients/majopela.yaml: 'name' is a required property
❌ Error in devices/ti-jacinto-j78s4xevm-01/ti-jacinto-j78s4xevm-01.yaml: the specified exporter host 'some-host' is not defined in configurations.
```

### Dry runs, useful to verify the configuration changes in merge requests

```shell
# the kubeconfig must contain cluster contexts with access to the jumpstarter controller
# specific resources like Exporters/Clients
export KUBECONFIG=~/.kube/config-from-secret

jumpstarter-lab-config apply --dry-run

Reading files from:
- clients/*.yaml
- devices/**/*.yaml
- exporter-templates/**/*.yaml
- locations/*.yaml
- jumpstarter-controllers/*.yaml

✅ All configurations are valid

Detected changes to apply:

2 new clients will be created: majopela, sdoherty.
1 new exporter will be created: ti-jacinto-j78s4xevm-01

⚠️ Warning: the following exporter will be deleted: ti-jacinto-j78s4xevm-44 if --prune is used

```

You can also specify a custom configuration file:
```shell
jumpstarter-lab-config apply --dry-run my-lab-config.yaml

Reading files from my-lab-config.yaml...

✅ All configurations are valid

Detected changes to apply:
...
```

### Real apply to targets

```shell
export KUBECONFIG=~/.kube/config-from-secret

jumpstarter-lab-config apply

Reading files from:
- clients/*.yaml
- devices/**/*.yaml
- exporter-templates/**/*.yaml
- locations/*.yaml
- jumpstarter-controllers/*.yaml

✅ All configurations are valid

Applying changes:

2 new clients will be created:

✅ Client 'majopela' created
✅ Client 'sdoherty' created

1 new exporter will be created:

✅ Exporter 'ti-jacinto-j78s4xevm-01' created

⚠️ Warning: there is a pending exporter to be deleted: ti-jacinto-j78s4xevm-44
            if you want to delete it, use --prune

```

You can also specify a custom configuration file:
```shell
jumpstarter-lab-config apply my-lab-config.yaml

Reading files from my-lab-config.yaml...

✅ All configurations are valid

Applying changes:
...
```


### Updating bootc images, useful to update bootc images in the exporter hosts (sidekicks)

```shell
export KUBECONFIG=~/.kube/config-from-secret

jumpstarter-lab-config apply

Reading files from:
- clients/*.yaml
- devices/**/*.yaml
- exporter-templates/**/*.yaml
- locations/*.yaml
- jumpstarter-controllers/*.yaml

✅ All configurations are valid

Applying changes:

3 exporter hosts will be updated with the new bootc image: quay.io/lab-management/exporter-bootc:0.7.1

✅ Exporter host 'ti-jacinto-j78s4xevm-01-sidekick' updated.
✅ Exporter host 'ti-jacinto-j78s4xevm-02-sidekick' updated.
⚠️ Exporter host 'ti-jacinto-j78s4xevm-44-sidekick' will update as soon as existing leases end.
```

## Design details

* We want this tool to be modular, will start by interfacing with the exporter-hosts via simple ssh
calls to update bootc images, restart target services, install updated configurations, etc.

* In the longer term we want to create a module to interface with flightctl/redhat edge management,
to better integrate with the Red Hat Edge Management ecosystem.

* We are choosing yaml in the kubernetes/crd format for several reasons:
   * It tracks version of the format, so we can evolve the configuration format over time.
   * One potential transition of this, could be a meta-operator that runs in an ACM cluster, providing
     a single point of management for all jumpstarter controllers in the enterprise, but not a goal
     now, just potential future evolution.

* We want to be able to use this tool in a gitops way, so we can have a single source of truth for
  the configuration of the jumpstarter controllers, and be able to apply changes in a controlled way,
  with dry-runs, linting, etc, yaml is not perfect, but it's human and git friendly.

* We are picking go as our programming language for the tool, as it has a good ecosystem for
  building command line tools, and we can use the kubernetes/client-go libraries to interface with the
  jumpstarter controllers and other kubernetes resources, also easing a possible future transition to
  a meta-operator.
