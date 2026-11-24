package rbac

var actionByLetter = map[rune]Action{
	'V': View,
	'C': Create,
	'E': Edit,
	'P': Publish,
	'R': Retire,
	'D': Delete,
}

func grant(letters string, sensitive ...Action) []Action {
	actions := make([]Action, 0, len(letters)+len(sensitive))
	for _, letter := range letters {
		actions = append(actions, actionByLetter[letter])
	}
	return append(actions, sensitive...)
}

var matrix = map[Object]map[Role][]Action{
	CosmeticItems: {
		SuperAdmin: grant("VCEPRD", CulturalReview),
		Content:    grant("VCEPRD", CulturalReview),
		LiveOps:    grant("V", SwitchOffForIncident),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	Assets: {
		SuperAdmin: grant("VCED"),
		Content:    grant("VCED"),
		LiveOps:    grant("V"),
	},
	RarityTable: {
		SuperAdmin: grant("VCEP"),
		Content:    grant("VCEP"),
		LiveOps:    grant("V"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	RewardBundles: liveOpsOwned("VCEPRD"),
	ShopOffers:    liveOpsOwned("VCEPRD"),
	StoreProducts: {
		SuperAdmin: grant("VCEPR", PushStorePrices),
		LiveOps:    grant("VCEPR"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	PassSeasons:        liveOpsOwned("VCEPRD"),
	Missions:           liveOpsOwned("VCEPRD"),
	Medals:             contentOwned("VCEPRD"),
	LeaderboardEmblems: contentOwned("VCEPR"),
	PromoCodes: {
		SuperAdmin: grant("VCEPRD", ApprovePromo),
		LiveOps:    grant("VCEPRD", ApprovePromo, DisablePromo),
		Support:    grant("V", EditTargetedAccounts),
		Analyst:    grant("", ViewCounts),
	},
	RankedSeasons: {
		SuperAdmin: grant("VCEPRD"),
		LiveOps:    grant("VCEPRD"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	LiveEvents: liveOpsOwned("VCEPRD"),
	GameConfig: liveOpsOwned("VCEP"),
	AntiCheatConfig: {
		SuperAdmin: grant("VCEP"),
	},
	PrivacyConfig: {
		SuperAdmin: grant("VCEP"),
	},
	ModerationConfig: {
		SuperAdmin: grant("VCEP"),
		Support:    grant("VCEP"),
	},
	GameStrings: {
		SuperAdmin: grant("VCEPRD"),
		Content:    grant("VCEPRD"),
		LiveOps:    grant("VCE"),
		Support:    grant("V"),
	},
	WordFilter: {
		SuperAdmin: grant("VCEPRD"),
		Support:    grant("VCEPRD"),
	},
	LiveControls: {
		SuperAdmin: grant("V", SwitchLiveControl),
		LiveOps:    grant("V", SwitchLiveControl),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	Snapshots: {
		SuperAdmin: grant("V", Rollback),
		Content:    grant("V", RollbackOwnDomains),
		LiveOps:    grant("V", RollbackOwnDomains),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	Players: {
		SuperAdmin: grant("V", ActOnPlayer),
		Support:    grant("V", ActOnPlayer),
	},
	Queues: {
		SuperAdmin: grant("V", ActOnQueue),
		Support:    grant("V", ActOnQueue),
	},
	RightsRequests: {
		SuperAdmin: grant("VE"),
		Support:    grant("VCE"),
	},
	ContentGateReports: {
		SuperAdmin: grant("VC"),
		Content:    grant("VC"),
		LiveOps:    grant("VC"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	Dashboards: {
		SuperAdmin: grant("V"),
		Content:    grant("V"),
		LiveOps:    grant("V"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	},
	AnalyticsEvents: {
		SuperAdmin: grant("V"),
		Analyst:    grant("V"),
	},
	AuditLog: {
		SuperAdmin: grant("V", ExportAudit),
		Content:    grant("V"),
		LiveOps:    grant("V"),
		Support:    grant("V"),
	},
	AdminAccounts: {
		SuperAdmin: grant("VCER", GrantSuperAdmin),
	},
}

func liveOpsOwned(letters string) map[Role][]Action {
	return map[Role][]Action{
		SuperAdmin: grant(letters),
		Content:    grant("V"),
		LiveOps:    grant(letters),
		Support:    grant("V"),
		Analyst:    grant("V"),
	}
}

func contentOwned(letters string) map[Role][]Action {
	return map[Role][]Action{
		SuperAdmin: grant(letters),
		Content:    grant(letters),
		LiveOps:    grant("V"),
		Support:    grant("V"),
		Analyst:    grant("V"),
	}
}
