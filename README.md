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


### Synchronizing access policies

Include `ExporterAccessPolicy` manifests through `sources.policies`:

```yaml
sources:
  policies:
    - policies/*.yaml
```

By default, `apply` creates and updates these policies in every configured Jumpstarter
instance's `spec.namespace` (required for policy sync), overriding any namespace
in the policy manifest. Specs and labels are synchronized; annotations are merged
and `managed-by: jumpstarter-lab-config` is added.

`apply --prune` also deletes policies absent from the configuration in those
namespaces, including policies created outside this tool. An empty policy set
therefore prunes all policies in the target namespaces. `--dry-run` reports
creations, updates, and requested deletions without writing changes. Client and
exporter filters do not filter policies, and `sync-clients: false` does not
disable policy sync.
If neither global nor instance-specific policy sources are configured, policy
synchronization and pruning are skipped.
Policies are processed in name order and per-policy errors are reported together.
All creates and updates are attempted, but pruning is skipped if any of them fail
so existing access policies remain available. Pruning attempts all stale policies.

### Per-instance clients and policies

Use the top-level `instances` section to opt individual instances into client
and policy synchronization. No enable/disable flags are needed:

```yaml
sources:
  jumpstarter_instances:
    - jumpstarter-instances/*.yaml

instances:
  production:
    clients:
      sources:
        - clients/common/*.yaml
    policies:
      sources:
        - policies/common/*.yaml
        - policies/production/*.yaml
  staging:
    # Clients are provisioned through OIDC; omit their sources to leave them alone.
    policies:
      sources:
        - policies/staging/*.yaml
```

Any other loaded instance is untouched for clients and policies because this
example has no global client or policy sources. Omitted sources skip all
synchronization and pruning, including Kubernetes reads, unless global sources
are inherited or synchronization is explicitly enabled.

Instance keys must match a loaded `JumpstarterInstance`'s `metadata.name`; unknown
names fail validation. Each resource type has two optional settings:

* `enabled`: an explicit value takes precedence; `false` skips all synchronization
  and pruning for that type on that instance. Otherwise, an explicit `sources`
  list (including `[]`) enables synchronization. With omitted instance sources,
  global sources enable synchronization unless legacy `spec.sync-clients: false`
  disables client synchronization. With no sources at either level, sync is skipped.
* `sources`: replaces the global sources for that instance and resource type.
  Omit it to inherit global sources. Paths are relative to the base configuration
  file. Include common globs explicitly when combining common and instance-specific
  resources. Duplicate resource names within a selected set are rejected; different
  instances may use different definitions of the same name.

An explicit `sources: []` at either level selects an empty set and enables sync;
it does **not** disable sync. This is distinct from omitting sources.
With sync enabled, `--prune` removes resources absent from that instance's selected
set, including resources created outside this tool. An empty set therefore prunes
all resources of that type in the target namespace. `--dry-run` previews changes,
and `--filter-clients` still limits client synchronization and pruning.

Both clients and policies are synchronized into the instance's `spec.namespace`,
which is required when their synchronization is enabled. This overrides any
namespace in the source manifests.

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
