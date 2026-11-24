package integrity

import (
	"slices"
	"testing"
)

func TestDecide(t *testing.T) {
	signer := []Reason{ReasonUnknownSigningCertificate}
	weak := []Reason{ReasonWeakDevice}
	both := []Reason{ReasonUnrecognizedApp, ReasonWeakDevice}
	policy := func(action Action, device DeviceRiskAction) Policy {
		return Policy{IntegrityFailureAction: action, IntegrityFailureWeight: 10, DeviceRiskAction: device, DeviceRiskWeight: 3}
	}
	integritySignal := Signal{Kind: SignalIntegrityFailure, Weight: 10, Reasons: signer}
	deviceSignal := Signal{Kind: SignalDeviceRisk, Weight: 3, Reasons: weak}
	cases := []struct {
		name    string
		policy  Policy
		purpose Purpose
		reasons []Reason
		action  Action
		signals []Signal
	}{
		{"pass", policy(ActionBlock, DeviceRiskBlockRanked), PurposeRankedMatch, nil, ActionAllow, nil},
		{"flag at session start", policy(ActionFlag, DeviceRiskFlag), PurposeSessionStart, signer, ActionFlag, []Signal{integritySignal}},
		{"flag at casual match", policy(ActionFlag, DeviceRiskFlag), PurposeCasualMatch, signer, ActionFlag, []Signal{integritySignal}},
		{"block at session start", policy(ActionBlock, DeviceRiskFlag), PurposeSessionStart, signer, ActionBlock, []Signal{integritySignal}},
		{"purchase always blocks", policy(ActionFlag, DeviceRiskFlag), PurposePurchase, signer, ActionBlock, []Signal{integritySignal}},
		{"ranked always blocks", policy(ActionFlag, DeviceRiskFlag), PurposeRankedMatch, signer, ActionBlock, []Signal{integritySignal}},
		{"unknown integrity action blocks", policy(ActionAllow, DeviceRiskFlag), PurposeSessionStart, signer, ActionBlock, []Signal{integritySignal}},
		{"device warn records no signal", policy(ActionFlag, DeviceRiskWarn), PurposeRankedMatch, weak, ActionWarn, nil},
		{"device flag", policy(ActionFlag, DeviceRiskFlag), PurposeRankedMatch, weak, ActionFlag, []Signal{deviceSignal}},
		{"device block ranked at ranked", policy(ActionFlag, DeviceRiskBlockRanked), PurposeRankedMatch, weak, ActionBlock, []Signal{deviceSignal}},
		{"device block ranked at casual", policy(ActionFlag, DeviceRiskBlockRanked), PurposeCasualMatch, weak, ActionFlag, []Signal{deviceSignal}},
		{"unknown device action blocks ranked", policy(ActionFlag, 0), PurposeRankedMatch, weak, ActionBlock, []Signal{deviceSignal}},
		{"most severe wins", policy(ActionFlag, DeviceRiskBlockRanked), PurposeRankedMatch, both, ActionBlock, []Signal{
			{Kind: SignalIntegrityFailure, Weight: 10, Reasons: []Reason{ReasonUnrecognizedApp}}, deviceSignal,
		}},
		{"warn does not lower a flag", policy(ActionFlag, DeviceRiskWarn), PurposeSessionStart, both, ActionFlag, []Signal{
			{Kind: SignalIntegrityFailure, Weight: 10, Reasons: []Reason{ReasonUnrecognizedApp}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := decide(tc.policy, tc.purpose, tc.reasons)
			if decision.Action != tc.action {
				t.Fatalf("action = %v, want %v", decision.Action, tc.action)
			}
			if decision.Allowed() != (tc.action != ActionBlock) {
				t.Fatalf("allowed = %v for %v", decision.Allowed(), decision.Action)
			}
			if !slices.EqualFunc(decision.Signals, tc.signals, equalSignal) {
				t.Fatalf("signals = %+v, want %+v", decision.Signals, tc.signals)
			}
		})
	}
}

func equalSignal(a, b Signal) bool {
	return a.Kind == b.Kind && a.Weight == b.Weight && slices.Equal(a.Reasons, b.Reasons)
}

func TestNamesAreStable(t *testing.T) {
	for reason := ReasonNonceMissing; reason <= ReasonSignatureInvalid; reason++ {
		if reason.String() == "invalid" {
			t.Errorf("reason %d has no name", reason)
		}
	}
	if Reason(0).String() != "invalid" || ReasonUnknownSigningCertificate.String() != "unknown_signing_certificate" {
		t.Fatal("reason names changed")
	}
	if PurposeRankedMatch.String() != "ranked_match" || ActionBlock.String() != "block" || SignalDeviceRisk.String() != "device_risk" || SignalHook.String() != "hook_signal" {
		t.Fatal("enum names changed")
	}
}
