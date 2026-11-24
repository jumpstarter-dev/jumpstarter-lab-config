package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	api "github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/config_lint"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/ssh"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/exporter/template"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/instance"
	"github.com/jumpstarter-dev/jumpstarter-lab-config/internal/templating"
	"github.com/spf13/cobra"
)

const (
	rotationClients   = "clients"
	rotationExporters = "exporters"
)

var rotateCredsCmd = &cobra.Command{
	Use:   "rotate-creds {exporters|clients|all} [config-file]",
	Short: "Rotate controller credentials and deliver exporter tokens over SSH",
	Long: `Rotate credentials for configured resources, one at a time.
Leased exporters are skipped by default; --wait rechecks them, and --force-leased
requires an exporter-name regexp and restarts matching leased exporters immediately.
Exporters are checked over SSH before rotation; their installed token is replaced
and otherwise a lease-aware restart is requested. Clients' new tokens remain in controller
Secrets for distribution to their users. Token values are never printed.`,
	ValidArgs:    []string{rotationExporters, rotationClients, "all"},
	Args:         cobra.MatchAll(cobra.RangeArgs(1, 2), validateRotationTarget),
	SilenceUsage: true,
	RunE:         runRotateCreds,
}

func validateRotationTarget(_ *cobra.Command, args []string) error {
	if !slices.Contains([]string{rotationExporters, rotationClients, "all"}, args[0]) {
		return fmt.Errorf("target must be exporters, clients, or all")
	}
	return nil
}

func runRotateCreds(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	opts := rotationOptions{dryRun: dryRun}
	opts.wait, _ = cmd.Flags().GetBool("wait")
	opts.waitTimeout, _ = cmd.Flags().GetDuration("wait-timeout")
	opts.waitInterval, _ = cmd.Flags().GetDuration("wait-interval")
	if opts.waitTimeout <= 0 || opts.waitInterval <= 0 {
		return fmt.Errorf("wait timeout and interval must be positive")
	}
	if cmd.Flags().Changed("force-leased") {
		pattern, _ := cmd.Flags().GetString("force-leased")
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("--force-leased requires a non-empty exporter-name regexp")
		}
		var err error
		opts.forceLeased, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid force-leased filter: %w", err)
		}
	}
	vaultPassFile, _ := cmd.Flags().GetString("vault-password-file")
	exporterPattern, _ := cmd.Flags().GetString("filter-exporters")
	clientPattern, _ := cmd.Flags().GetString("filter-clients")
	exporterFilter, err := regexp.Compile(exporterPattern)
	if err != nil {
		return fmt.Errorf("invalid exporter filter: %w", err)
	}
	clientFilter, err := regexp.Compile(clientPattern)
	if err != nil {
		return fmt.Errorf("invalid client filter: %w", err)
	}
	configFile := defaultConfigFile
	if len(args) == 2 {
		configFile = args[1]
	}
	cfg, err := config.LoadConfig(configFile, vaultPassFile)
	if err != nil {
		return err
	}
	if err := config_lint.ValidateWithError(cfg); err != nil {
		return err
	}
	tapplier, err := templating.NewTemplateApplier(cfg, nil)
	if err != nil {
		return err
	}
	var failures []error
	for _, name := range slices.Sorted(maps.Keys(cfg.Loaded.JumpstarterInstances)) {
		instConfig := cfg.Loaded.JumpstarterInstances[name].DeepCopy()
		if err := tapplier.Apply(instConfig); err != nil {
			return err
		}
		inst, err := instance.NewInstance(instConfig, instConfig.Spec.Kubeconfig, dryRun, false, false)
		if err != nil {
			return err
		}
		if args[0] != rotationClients {
			failures = append(failures, rotateExporters(cmd, cfg, tapplier, inst, exporterFilter, opts))
		}
		if args[0] != rotationExporters {
			failures = append(failures, rotateClients(cmd.Context(), cfg, inst, clientFilter, dryRun))
		}
	}
	return errors.Join(failures...)
}

func rotateExporters(cmd *cobra.Command, cfg *config.Config, tapplier *templating.TemplateApplier,
	inst *instance.Instance, filter *regexp.Regexp, opts rotationOptions) error {
	var names []string
	out := cmd.OutOrStdout()
	for _, name := range slices.Sorted(maps.Keys(cfg.Loaded.ExporterInstances)) {
		exporter := cfg.Loaded.ExporterInstances[name]
		if exporter == nil || exporter.Spec.JumpstarterInstanceRef.Name != inst.GetConfig().Name ||
			!filter.MatchString(name) {
			continue
		}
		dead, _ := exporter.IsDead()
		unmanaged, _ := exporter.IsUnmanaged()
		if dead || unmanaged || exporter.Spec.ExporterHostRef.Name == "" {
			_, _ = fmt.Fprintf(out, "⏸️  [%s] Skipping exporter %s: dead, unmanaged, or no SSH host\n",
				inst.GetConfig().Name, name)
			continue
		}
		names = append(names, name)
	}
	return rotateExporterBatch(cmd.Context(), inst.GetClient(), inst.GetConfig().Spec.Namespace, names, opts, out,
		func(ctx context.Context, name string, force bool) error {
			exporter := cfg.Loaded.ExporterInstances[name]
			if err := rotateExporter(ctx, cfg, tapplier, inst, exporter, opts.dryRun, force); err != nil {
				return err
			}
			switch {
			case opts.dryRun:
				_, _ = fmt.Fprintf(out, "📄 [%s] Would rotate exporter %s (SSH and controller checks passed)\n",
					inst.GetConfig().Name, name)
			case force:
				_, _ = fmt.Fprintf(out, "✅ [%s] Exporter %s token installed; immediate restart completed\n",
					inst.GetConfig().Name, name)
			default:
				_, _ = fmt.Fprintf(out, "✅ [%s] Exporter %s token installed; lease-aware restart requested\n",
					inst.GetConfig().Name, name)
			}
			return nil
		})
}

func rotateExporter(ctx context.Context, cfg *config.Config, tapplier *templating.TemplateApplier,
	inst *instance.Instance, exporter *api.ExporterInstance, dryRun, forceRestart bool) error {
	host := cfg.Loaded.ExporterHosts[exporter.Spec.ExporterHostRef.Name]
	if host == nil {
		return fmt.Errorf("exporter host %s not found", exporter.Spec.ExporterHostRef.Name)
	}
	renderedHost := host.DeepCopy()
	if err := tapplier.Apply(renderedHost); err != nil {
		return err
	}
	et, err := template.NewExporterInstanceTemplater(cfg, exporter)
	if err != nil {
		return err
	}
	rendered, err := et.RenderTemplateConfig()
	if err != nil {
		return err
	}
	hostSSH, err := ssh.NewSSHHostManager(renderedHost)
	if err != nil {
		return fmt.Errorf("SSH preflight failed; controller credential unchanged: %w", err)
	}
	defer func() { _ = hostSSH.Close() }()
	rotated := false
	err = hostSSH.RotateCredentials(rendered.Spec.ExporterMetadata.Name, dryRun, forceRestart, func() (string, error) {
		secret, err := inst.RotateCredentials(ctx, "exporter", exporter.Name)
		if err != nil {
			return "", err
		}
		rotated = !dryRun
		return string(secret.Data["token"]), nil
	})
	if err != nil && rotated {
		return fmt.Errorf("controller credential rotated but host update incomplete; "+
			"retry this exporter or use apply to deliver its current credential: %w", err)
	}
	return err
}

func rotateClients(ctx context.Context, cfg *config.Config, inst *instance.Instance,
	filter *regexp.Regexp, dryRun bool) error {
	name := inst.GetConfig().Name
	if !cfg.Instances[name].Clients.IsEnabled(cfg.Sources.Clients, inst.GetConfig().Spec.SyncClients) {
		return nil
	}
	var failures []error
	for _, clientName := range slices.Sorted(maps.Keys(cfg.Loaded.GetClientsForInstance(name))) {
		if !filter.MatchString(clientName) {
			continue
		}
		secret, err := inst.RotateCredentials(ctx, "client", clientName)
		if err != nil {
			failure := fmt.Errorf("[%s] client %s: %w", name, clientName, err)
			fmt.Printf("❌ %s\n", failure)
			failures = append(failures, failure)
			continue
		}
		if dryRun {
			fmt.Printf("📄 [%s] Would rotate client %s\n", name, clientName)
		} else {
			fmt.Printf("✅ [%s] Client %s rotated; retrieve replacement from Secret %s/%s\n",
				name, clientName, secret.Namespace, secret.Name)
		}
	}
	return errors.Join(failures...)
}

func init() {
	rotateCredsCmd.Flags().Bool("dry-run", false,
		"Check exporter SSH access and preview rotation without changing credentials or files")
	rotateCredsCmd.Flags().String("vault-password-file", "", "Path to the vault password file for decrypting variables")
	rotateCredsCmd.Flags().String("filter-clients", "", "Regexp pattern to filter clients by name")
	rotateCredsCmd.Flags().String("filter-exporters", "", "Regexp pattern to filter exporters by name")
	rotateCredsCmd.Flags().Bool("wait", false, "Wait for leased exporters to become free (dry runs never wait)")
	rotateCredsCmd.Flags().Duration("wait-timeout", 30*time.Minute, "Maximum wait per instance, including rotations")
	rotateCredsCmd.Flags().Duration("wait-interval", 10*time.Second, "Interval between fresh lease checks")
	rotateCredsCmd.Flags().String("force-leased", "",
		"Regexp selecting leased exporters to rotate and immediately restart")
	rootCmd.AddCommand(rotateCredsCmd)
}
