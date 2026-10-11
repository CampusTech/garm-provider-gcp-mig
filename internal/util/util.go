// SPDX-License-Identifier: Apache-2.0
//
// Adapted from github.com/cloudbase/garm-provider-gcp (Apache-2.0).

// Package util converts GCE instances to GARM provider instances.
package util

import (
	"fmt"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/cloudbase/garm-provider-common/params"
	"github.com/ut0mt8/garm-provider-gcp-mig/internal/spec"
)

const MetadataRunnerName = "runner_name"

// InstanceName normalizes a runner name to a valid GCE instance name.
func InstanceName(name string) string {
	return strings.ToLower(name)
}

// ProviderID is "<zone>/<name>", the same format as upstream regional placement.
func ProviderID(zone, name string) string {
	return zone + "/" + InstanceName(name)
}

// SplitProviderID splits "<zone>/<name>".
func SplitProviderID(providerID string) (zone, name string, ok bool) {
	zone, name, ok = strings.Cut(providerID, "/")
	if !ok || zone == "" || name == "" || strings.Contains(name, "/") {
		return "", "", false
	}
	return strings.ToLower(zone), InstanceName(name), true
}

// ZoneFromURL extracts the zone from ".../zones/<zone>[/...]".
func ZoneFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "zones" {
			return parts[i+1]
		}
	}
	return ""
}

func metadataValue(inst *computepb.Instance, key string) (string, bool) {
	for _, item := range inst.GetMetadata().GetItems() {
		if item.GetKey() == key {
			return item.GetValue(), true
		}
	}
	return "", false
}

// ToProviderInstance converts a GCE instance into what GARM expects.
func ToProviderInstance(inst *computepb.Instance) (params.ProviderInstance, error) {
	if inst == nil || inst.Name == nil {
		return params.ProviderInstance{}, fmt.Errorf("instance or instance name is nil")
	}
	zone := ZoneFromURL(inst.GetZone())
	if zone == "" {
		return params.ProviderInstance{}, fmt.Errorf("instance %s has no zone", inst.GetName())
	}
	name := inst.GetName()
	if runnerName, ok := metadataValue(inst, MetadataRunnerName); ok && runnerName != "" {
		name = runnerName // GARM's original casing
	}

	out := params.ProviderInstance{
		ProviderID: ProviderID(zone, inst.GetName()),
		Name:       name,
		OSType:     params.OSType(inst.GetLabels()[spec.LabelOSType]),
	}
	if disks := inst.GetDisks(); len(disks) > 0 {
		switch strings.ToUpper(disks[0].GetArchitecture()) {
		case "ARM64":
			out.OSArch = params.Arm64
		default: // X86_64, or unset on some images
			out.OSArch = params.Amd64
		}
	}
	switch inst.GetStatus() {
	case "RUNNING", "STAGING", "PROVISIONING":
		out.Status = params.InstanceRunning
	case "STOPPING", "TERMINATED", "SUSPENDING", "SUSPENDED":
		out.Status = params.InstanceStopped
	default:
		out.Status = params.InstanceStatusUnknown
	}
	return out, nil
}
