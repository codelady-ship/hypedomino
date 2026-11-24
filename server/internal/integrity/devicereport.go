package integrity

import (
	"context"
	"slices"
)

const PurposeClientReport = "client_report"

var (
	clientDeviceSignals = []string{"root", "test_keys", "emulator", "jailbreak"}
	clientHookSignals   = []string{"instrumentation", "hook_framework", "debugger"}
)

type DeviceReports struct {
	policy  Policy
	signals SignalSink
}

func NewDeviceReports(policy Policy, signals SignalSink) *DeviceReports {
	return &DeviceReports{policy: policy, signals: signals}
}

func (d *DeviceReports) Observe(ctx context.Context, account string, deviceNames, hookNames []string) {
	d.observeDevice(ctx, account, deviceNames)
	d.observeHooks(ctx, account, hookNames)
}

func (d *DeviceReports) observeDevice(ctx context.Context, account string, names []string) {
	known := knownSignals(clientDeviceSignals, names)
	if len(known) == 0 || d.policy.DeviceRiskAction == DeviceRiskWarn {
		return
	}
	d.signals.RecordIntegrity(ctx, account, PurposeClientReport, SignalDeviceRisk.String(), d.policy.DeviceRiskWeight, known)
}

func (d *DeviceReports) observeHooks(ctx context.Context, account string, names []string) {
	known := knownSignals(clientHookSignals, names)
	if len(known) == 0 {
		return
	}
	d.signals.RecordIntegrity(ctx, account, PurposeClientReport, SignalHook.String(), d.policy.HookSignalWeight, known)
}

func knownSignals(allowed, names []string) []string {
	var known []string
	for _, signal := range allowed {
		if slices.Contains(names, signal) {
			known = append(known, signal)
		}
	}
	return known
}
