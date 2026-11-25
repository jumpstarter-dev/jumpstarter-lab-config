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


### Rotating credentials

```shell
jumpstarter-lab-config rotate-creds exporters --dry-run
jumpstarter-lab-config rotate-creds all --legacy-only --dry-run
jumpstarter-lab-config rotate-creds exporters --filter-exporters '^board-01$'
jumpstarter-lab-config rotate-creds exporters --wait --wait-timeout 30m
jumpstarter-lab-config rotate-creds exporters --force-leased '^board-01$' --dry-run
jumpstarter-lab-config rotate-creds clients --filter-clients '^alice$'
jumpstarter-lab-config rotate-creds all my-lab-config.yaml
```

`rotate-creds` processes configured resources sequentially, with the same config
file, vault password, and name-filter conventions as `apply`. For each exporter:

1. Check its current lease. Leased exporters are skipped by default; their lease,
   owner/shared holders, and current end time are printed for coordination.
2. Connect to its configured host over SSH/SFTP and check the installed exporter
   YAML, write access, and systemd unit. A failed preflight leaves its controller
   credential unchanged.
3. Delete its credential Secret and wait up to 90 seconds for the controller to
   regenerate it. The deletion uses UID/resource-version preconditions, and the
   replacement must belong to the same exporter.
4. Atomically replace only the token in `/etc/jumpstarter/exporters/<service>.yaml`,
   preserving the other configuration values, comments, ownership and permissions.
5. Request a lease-aware restart with SIGHUP, as `apply` does, or restart an
   inactive service. SIGHUP covers a lease starting between the check and deletion.
   Forced leased rotations use an immediate `systemctl restart` instead.

`--legacy-only` selects only credential Secrets with a nonempty
`jumpstarter.dev/legacy-token-hash` annotation, for exporters, clients, or `all`.
Resources without that annotation (or without internal credentials) are reported
as skipped. Exporters are filtered before lease waiting and SSH access, and the
annotation is checked again before deletion using the same UID/resource-version
preconditions. It combines with name filters, `--wait`, `--force-leased`, and
`--dry-run`; forcing does not override this filter. The annotation is the selection
criterion, not a check of the installed token. If an earlier rotation generated a
new unannotated Secret but delivery failed, use filtered `apply` to deliver it.

`--wait` rotates free exporters first and rechecks skipped exporters every
`--wait-interval` (default 10s), rotating each once it becomes free. The
`--wait-timeout` (default 30m) bounds the operation per instance, including rotations;
timeout exits nonzero and names the pending exporters. In-flight SSH operations
finish before the timeout is reported. Lease end times are read
again on every pass, including extensions; they are informational, not a schedule
for deciding when an exporter is free.

`--force-leased <regexp>` explicitly selects leased exporters for immediate
rotation/restart, interrupting their sessions. It is intersected with
`--filter-exporters`; nonmatching leased exporters remain skipped (or waited for
with `--wait`). Holders are printed before each forced rotation. Run `--dry-run`
first to see the same holder/end-time table and notify users. Dry runs never wait
or restart anything, even with `--wait` or `--force-leased`.

The service name comes from the exporter's rendered config template. Dead,
unmanaged, and hostless exporters are skipped. Failures are reported per resource;
the command continues with the others and exits nonzero if any failed. If rotation
succeeded but installation or restart failed, retry that exporter, or use
`apply --filter-exporters '^name$'` to deliver the current controller credential.
No bootc or container-image updates are performed by `rotate-creds`.
If a bootc upgrade is in progress, exporter rotation fails preflight before
changing the controller credential, rather than installing a token whose restart
would have to be deferred. This check also applies to forced restarts and dry runs.

`clients` rotates only clients selected for synchronization on each instance.
Client machines have no SSH destinations in the lab configuration, so replacements
remain in the controller Secrets; the command prints their names for distribution
to users (for example through `jmp admin import client <name> --namespace <namespace>
--out client.yaml`, using the appropriate cluster context). Token values are never
printed. Resources without internal credential Secrets cannot be rotated this way.

During signing-key migration, an internal-token client loses access as soon as
its Secret is deleted and stays locked out until its user imports the replacement.
OIDC client authentication is unaffected. **`rotate-creds all` without filters
rotates every selected internal-token client at once from the users' perspective.**
Coordinate replacement distribution with users first, and use `--filter-clients`
to migrate a limited group at a time.

`--dry-run` performs exporter SSH/read-access checks and verifies the controller
credential Secret and its ownership, without changing remote files or credentials.
It also verifies existing client credential Secrets and their ownership.
Rotation requires Kubernetes get/delete access to credential Secrets and get access
to their Client/Exporter resources, plus a running controller to regenerate them.
Exporter hosts need SFTP's POSIX rename extension and permission to update their
config files and signal their services.

Deletion immediately revokes old-key credentials that rely on the Secret's
`jumpstarter.dev/legacy-token-hash` allowance during a signing-key migration.
Revocation happens when the Secret is deleted, **before** the replacement is
installed. Tokens whose signatures remain valid with the current key may
continue to authenticate until expiry.

The 90-second regeneration wait requires a running, reconciling controller. If it
times out, the old Secret is already deleted and its legacy allowance is gone;
restore controller reconciliation and deliver the replacement with `apply` (or
import it for clients). Regeneration timing has not been verified against a live
lab by this change's automated tests.

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

### Managing exporter hosts through Flight Control

An exporter host can be managed through [Flight Control](https://github.com/flightctl/flightctl)
instead of SSH. Select its device by label in the `ExporterHost`:

```yaml
apiVersion: meta.jumpstarter.dev/v1alpha1
kind: ExporterHost
metadata:
  name: ti-jacinto-j784s4xevm-01-sidekick-01
spec:
  management:
    flightctl:
      deviceSelector:
        host: "$( name )"
```

The selector must match exactly one device. Flight Control names devices after their key
fingerprint, so approve the enrollment with a label, for example
`flightctl approve -l host=ti-jacinto-j784s4xevm-01-sidekick-01 er/<name>`. A host managed through
Flight Control has no `management.ssh`: lint rejects a host that sets both.

When a run processes such a host, `apply` talks to the Flight Control API with the client config
written by `flightctl login`
(`--flightctl-config` selects another file). It puts the quadlet or systemd unit and the exporter
config of every exporter on the host into one inline config provider named
`jumpstarter-exporters` in the device spec. Other config providers, the OS image and the
applications of the device are left alone, and so are files of exporters that this run does not
process. `--dry-run` shows the files that would change.

What differs from SSH:

* `jumpstarter-lab-config` does not restart anything. The Flight Control agent writes the files,
  and the host image must carry an `afterUpdating` hook that reloads systemd and starts the
  exporter or sends it `SIGHUP`, so that a leased exporter restarts once the lease ends.
* The device must not be owned by a fleet: Flight Control rejects spec changes for such a device.
* `apply` reads the device and patches it only if it has not changed since. When something else
  changed the device in between, the patch fails and `apply` has to run again.
* No other config provider of the device may deliver the same files: Flight Control rejects a
  spec in which two providers write the same path.
* The exporter container image is not compared with the registry. Pin a tag or digest in the
  template to roll out a new image.
* The OS image is not managed yet.
* The Flight Control agent keeps writing the exporter config as it is in the device spec. An
  exporter that renews its token writes the new one to the same file and the agent overwrites it
  again, so run `apply` regularly to bring the current token back into the spec.

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
