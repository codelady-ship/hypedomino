package integrity

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestClientDeviceReportsOnlyRaiseRisk(t *testing.T) {
	flag := Policy{DeviceRiskAction: DeviceRiskFlag, DeviceRiskWeight: 5}
	blockRanked := Policy{DeviceRiskAction: DeviceRiskBlockRanked, DeviceRiskWeight: 5}
	warn := Policy{DeviceRiskAction: DeviceRiskWarn, DeviceRiskWeight: 5}
	cases := []struct {
		name   string
		policy Policy
		names  []string
		want   []recordedSignal
	}{
		{"no signals", flag, nil, nil},
		{"unknown names are ignored", flag, []string{"instrumentation", strings.Repeat("x", 4096)}, nil},
		{"known names are recorded once in a fixed order", flag, []string{"jailbreak", "root", "root", "hook_framework"},
			[]recordedSignal{{"account-1", PurposeClientReport, "device_risk", 5, []string{"root", "jailbreak"}}}},
		{"block ranked records the signal and blocks nothing", blockRanked, []string{"emulator", "test_keys"},
			[]recordedSignal{{"account-1", PurposeClientReport, "device_risk", 5, []string{"test_keys", "emulator"}}}},
		{"warn records nothing", warn, []string{"root"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &signalLog{}
			NewDeviceReports(c.policy, log).Observe(context.Background(), "account-1", c.names, nil)
			if !reflect.DeepEqual(log.signals, c.want) {
				t.Fatalf("recorded %v, want %v", log.signals, c.want)
			}
		})
	}
}

func TestClientHookReportsAreRecordedWithTheHookWeight(t *testing.T) {
	weighted := Policy{DeviceRiskAction: DeviceRiskFlag, DeviceRiskWeight: 5, HookSignalWeight: 8}
	unweighted := Policy{DeviceRiskAction: DeviceRiskFlag, DeviceRiskWeight: 5}
	warn := Policy{DeviceRiskAction: DeviceRiskWarn, DeviceRiskWeight: 5, HookSignalWeight: 8}
	cases := []struct {
		name   string
		policy Policy
		device []string
		hooks  []string
		want   []recordedSignal
	}{
		{"no hook signals", weighted, nil, nil, nil},
		{"unknown and device names are ignored", weighted, nil, []string{"root", strings.Repeat("x", 4096)}, nil},
		{"known names are recorded once in a fixed order", weighted, nil, []string{"debugger", "instrumentation", "debugger", "hook_framework"},
			[]recordedSignal{{"account-1", PurposeClientReport, "hook_signal", 8, []string{"instrumentation", "hook_framework", "debugger"}}}},
		{"an unset weight still records the evidence", unweighted, nil, []string{"debugger"},
			[]recordedSignal{{"account-1", PurposeClientReport, "hook_signal", 0, []string{"debugger"}}}},
		{"the device risk action does not silence hooks", warn, []string{"root"}, []string{"instrumentation"},
			[]recordedSignal{{"account-1", PurposeClientReport, "hook_signal", 8, []string{"instrumentation"}}}},
		{"device and hook signals are recorded apart", weighted, []string{"emulator"}, []string{"hook_framework"},
			[]recordedSignal{
				{"account-1", PurposeClientReport, "device_risk", 5, []string{"emulator"}},
				{"account-1", PurposeClientReport, "hook_signal", 8, []string{"hook_framework"}},
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &signalLog{}
			NewDeviceReports(c.policy, log).Observe(context.Background(), "account-1", c.device, c.hooks)
			if !reflect.DeepEqual(log.signals, c.want) {
				t.Fatalf("recorded %v, want %v", log.signals, c.want)
			}
		})
	}
}
