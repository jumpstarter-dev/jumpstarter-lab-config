package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	js "github.com/jumpstarter-dev/jumpstarter/controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type rotationOptions struct {
	dryRun, wait              bool
	waitInterval, waitTimeout time.Duration
	forceLeased               *regexp.Regexp
}

// Process free exporters first, then recheck only those skipped because of a lease.
// rotate keeps SSH delivery separate from lease selection so every pass uses fresh API state.
func rotateExporterBatch(ctx context.Context, reader client.Reader, namespace string, names []string,
	opts rotationOptions, out io.Writer, rotate func(context.Context, string, bool) error) error {
	var failures []error
	pending := names
	headerPrinted := false
	check := func(ctx context.Context) (bool, error) {
		var waiting []string
		for index, name := range pending {
			if err := ctx.Err(); err != nil {
				pending = append(waiting, pending[index:]...)
				return false, err
			}
			lease, err := rotationLease(ctx, reader, client.ObjectKey{Namespace: namespace, Name: name})
			force := lease != nil && opts.forceLeased != nil && opts.forceLeased.MatchString(name)
			if err == nil && lease != nil {
				if !headerPrinted {
					_, _ = fmt.Fprintf(out, "%-32s %-24s %-32s %-25s %s\n", "EXPORTER", "LEASE", "HOLDERS", "CURRENT END", "ACTION")
					headerPrinted = true
				}
				action := "skip leased"
				if force {
					action = "FORCE immediate restart"
				} else if opts.wait {
					action = "wait"
				}
				if opts.dryRun {
					action = "would " + action
				}
				holders, end := rotationLeaseDetails(lease)
				_, _ = fmt.Fprintf(out, "%-32s %-24s %-32s %-25s %s\n", namespace+"/"+name, lease.GetName(), holders, end, action)
				if !force {
					waiting = append(waiting, name)
					continue
				}
			}
			if err == nil {
				err = rotate(ctx, name, force)
			}
			if err != nil {
				failure := fmt.Errorf("exporter %s/%s: %w", namespace, name, err)
				_, _ = fmt.Fprintf(out, "❌ %s\n", failure)
				failures = append(failures, failure)
			}
		}
		pending = waiting
		return len(pending) == 0, nil
	}
	var err error
	if opts.wait && !opts.dryRun {
		err = wait.PollUntilContextTimeout(ctx, opts.waitInterval, opts.waitTimeout, true, check)
	} else {
		_, err = check(ctx)
	}
	if err != nil {
		failures = append(failures, fmt.Errorf("waiting for exporters %s: %w", strings.Join(pending, ", "), err))
	}
	return errors.Join(failures...)
}

func rotationLease(ctx context.Context, reader client.Reader,
	key client.ObjectKey) (*unstructured.Unstructured, error) {
	exporter := &js.Exporter{}
	if err := reader.Get(ctx, key, exporter); err != nil {
		return nil, err
	}
	if exporter.Status.LeaseRef == nil {
		return nil, nil
	}
	// The pinned controller API predates sharedWith. Read the Lease unstructured
	// so newer controllers' shared holders are included without a dependency bump.
	lease := &unstructured.Unstructured{}
	lease.SetGroupVersionKind(js.GroupVersion.WithKind("Lease"))
	key.Name = exporter.Status.LeaseRef.Name
	if err := reader.Get(ctx, key, lease); err != nil {
		return nil, fmt.Errorf("read assigned lease %s: %w", key.Name, err)
	}
	return lease, nil
}

func rotationLeaseDetails(lease *unstructured.Unstructured) (string, string) {
	owner, _, _ := unstructured.NestedString(lease.Object, "spec", "clientRef", "name")
	shared, _, _ := unstructured.NestedStringSlice(lease.Object, "spec", "sharedWith")
	holders := strings.Join(append([]string{owner}, shared...), ", ")
	if owner == "" {
		holders = "unknown" + holders
	}
	end, _, _ := unstructured.NestedString(lease.Object, "spec", "endTime")
	if end != "" {
		return holders, end
	}
	begin, _, _ := unstructured.NestedString(lease.Object, "status", "beginTime")
	duration, _, _ := unstructured.NestedString(lease.Object, "spec", "duration")
	start, startErr := time.Parse(time.RFC3339, begin)
	span, spanErr := time.ParseDuration(duration)
	if startErr == nil && spanErr == nil {
		return holders, start.Add(span).UTC().Format(time.RFC3339)
	}
	return holders, "unknown"
}
