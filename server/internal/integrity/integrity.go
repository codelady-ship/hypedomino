package integrity

import "time"

type Clock interface {
	Now() time.Time
}

type Purpose uint8

const (
	PurposeSessionStart Purpose = iota + 1
	PurposePurchase
	PurposeRankedMatch
	PurposeCasualMatch
)

func (p Purpose) String() string {
	switch p {
	case PurposeSessionStart:
		return "session_start"
	case PurposePurchase:
		return "purchase"
	case PurposeRankedMatch:
		return "ranked_match"
	case PurposeCasualMatch:
		return "casual_match"
	}
	return "invalid"
}

func (p Purpose) valid() bool {
	return p >= PurposeSessionStart && p <= PurposeCasualMatch
}

func (p Purpose) alwaysBlocksIntegrityFailure() bool {
	return p == PurposePurchase || p == PurposeRankedMatch
}

type Binding struct {
	Account string
	Session string
	Purpose Purpose
}

type Reason uint8

const (
	ReasonNonceMissing Reason = iota + 1
	ReasonNonceUnknown
	ReasonNonceExpired
	ReasonNonceBindingMismatch
	ReasonTokenRejected
	ReasonTokenMalformed
	ReasonTokenStale
	ReasonPackageMismatch
	ReasonUnrecognizedApp
	ReasonUnknownSigningCertificate
	ReasonUnlicensed
	ReasonWeakDevice
	ReasonAttestationMalformed
	ReasonCertificateChain
	ReasonChallengeMismatch
	ReasonKeyIDMismatch
	ReasonAppIDMismatch
	ReasonCounterInvalid
	ReasonCounterReplay
	ReasonEnvironmentMismatch
	ReasonSignatureInvalid
)

var reasonNames = map[Reason]string{
	ReasonNonceMissing:              "nonce_missing",
	ReasonNonceUnknown:              "nonce_unknown",
	ReasonNonceExpired:              "nonce_expired",
	ReasonNonceBindingMismatch:      "nonce_binding_mismatch",
	ReasonTokenRejected:             "token_rejected",
	ReasonTokenMalformed:            "token_malformed",
	ReasonTokenStale:                "token_stale",
	ReasonPackageMismatch:           "package_mismatch",
	ReasonUnrecognizedApp:           "unrecognized_app",
	ReasonUnknownSigningCertificate: "unknown_signing_certificate",
	ReasonUnlicensed:                "unlicensed",
	ReasonWeakDevice:                "weak_device",
	ReasonAttestationMalformed:      "attestation_malformed",
	ReasonCertificateChain:          "certificate_chain",
	ReasonChallengeMismatch:         "challenge_mismatch",
	ReasonKeyIDMismatch:             "key_id_mismatch",
	ReasonAppIDMismatch:             "app_id_mismatch",
	ReasonCounterInvalid:            "counter_invalid",
	ReasonCounterReplay:             "counter_replay",
	ReasonEnvironmentMismatch:       "environment_mismatch",
	ReasonSignatureInvalid:          "signature_invalid",
}

func (r Reason) String() string {
	if name, known := reasonNames[r]; known {
		return name
	}
	return "invalid"
}

func appendReason(reasons []Reason, reason Reason) []Reason {
	if reason == 0 {
		return reasons
	}
	return append(reasons, reason)
}

type Action uint8

const (
	ActionAllow Action = iota
	ActionWarn
	ActionFlag
	ActionBlock
)

func (a Action) String() string {
	switch a {
	case ActionAllow:
		return "allow"
	case ActionWarn:
		return "warn"
	case ActionFlag:
		return "flag"
	case ActionBlock:
		return "block"
	}
	return "invalid"
}

type DeviceRiskAction uint8

const (
	DeviceRiskWarn DeviceRiskAction = iota + 1
	DeviceRiskFlag
	DeviceRiskBlockRanked
)

type DeviceLevel uint8

const (
	DeviceLevelNone DeviceLevel = iota
	DeviceLevelBasic
	DeviceLevelDevice
	DeviceLevelStrong
)

type Policy struct {
	IntegrityFailureAction Action
	IntegrityFailureWeight int
	DeviceRiskAction       DeviceRiskAction
	DeviceRiskWeight       int
	HookSignalWeight       int
	MinDeviceLevel         DeviceLevel
	RequireLicensed        bool
}

type SignalKind uint8

const (
	SignalIntegrityFailure SignalKind = iota + 1
	SignalDeviceRisk
	SignalHook
)

func (k SignalKind) String() string {
	switch k {
	case SignalIntegrityFailure:
		return "integrity_failure"
	case SignalDeviceRisk:
		return "device_risk"
	case SignalHook:
		return "hook_signal"
	}
	return "invalid"
}

type Signal struct {
	Kind    SignalKind
	Weight  int
	Reasons []Reason
}

type Decision struct {
	Purpose Purpose
	Reasons []Reason
	Action  Action
	Signals []Signal
}

func (d Decision) Passed() bool {
	return len(d.Reasons) == 0
}

func (d Decision) Allowed() bool {
	return d.Action != ActionBlock
}

func decide(policy Policy, purpose Purpose, reasons []Reason) Decision {
	decision := Decision{Purpose: purpose, Reasons: reasons, Action: ActionAllow}
	integrityReasons, deviceReasons := splitReasons(reasons)
	if len(integrityReasons) > 0 {
		decision.raise(integrityFailureAction(policy, purpose))
		decision.Signals = append(decision.Signals, Signal{Kind: SignalIntegrityFailure, Weight: policy.IntegrityFailureWeight, Reasons: integrityReasons})
	}
	if len(deviceReasons) > 0 {
		action := deviceRiskAction(policy, purpose)
		decision.raise(action)
		if action != ActionWarn {
			decision.Signals = append(decision.Signals, Signal{Kind: SignalDeviceRisk, Weight: policy.DeviceRiskWeight, Reasons: deviceReasons})
		}
	}
	return decision
}

func (d *Decision) raise(action Action) {
	if action > d.Action {
		d.Action = action
	}
}

func splitReasons(reasons []Reason) (integrityReasons, deviceReasons []Reason) {
	for _, reason := range reasons {
		if reason == ReasonWeakDevice {
			deviceReasons = append(deviceReasons, reason)
			continue
		}
		integrityReasons = append(integrityReasons, reason)
	}
	return integrityReasons, deviceReasons
}

func integrityFailureAction(policy Policy, purpose Purpose) Action {
	if purpose.alwaysBlocksIntegrityFailure() {
		return ActionBlock
	}
	if policy.IntegrityFailureAction == ActionFlag {
		return ActionFlag
	}
	return ActionBlock
}

func deviceRiskAction(policy Policy, purpose Purpose) Action {
	switch policy.DeviceRiskAction {
	case DeviceRiskWarn:
		return ActionWarn
	case DeviceRiskFlag:
		return ActionFlag
	}
	if purpose == PurposeRankedMatch {
		return ActionBlock
	}
	return ActionFlag
}

func (d Decision) ReasonNames() []string {
	names := make([]string, len(d.Reasons))
	for i, reason := range d.Reasons {
		names[i] = reason.String()
	}
	return names
}

func ParseAction(name string) (Action, bool) {
	switch name {
	case "flag":
		return ActionFlag, true
	case "block":
		return ActionBlock, true
	}
	return 0, false
}

func ParseDeviceRiskAction(name string) (DeviceRiskAction, bool) {
	switch name {
	case "warn":
		return DeviceRiskWarn, true
	case "flag":
		return DeviceRiskFlag, true
	case "block_ranked":
		return DeviceRiskBlockRanked, true
	}
	return 0, false
}

func ParseDeviceLevel(name string) (DeviceLevel, bool) {
	switch name {
	case "none":
		return DeviceLevelNone, true
	case "basic":
		return DeviceLevelBasic, true
	case "device":
		return DeviceLevelDevice, true
	case "strong":
		return DeviceLevelStrong, true
	}
	return 0, false
}
