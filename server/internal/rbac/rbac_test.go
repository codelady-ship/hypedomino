package rbac

import (
	"os"
	"slices"
	"strings"
	"testing"
)

const adminSpec = "../../../docs/admin.md"

var specColumns = []Role{SuperAdmin, Content, LiveOps, Support, Analyst}

var objectBySpecRow = map[string]Object{
	"Cosmetic items (5.1)":                        CosmeticItems,
	"Assets (5.2)":                                Assets,
	"Rarity table (5.3)":                          RarityTable,
	"Reward bundles (5.4)":                        RewardBundles,
	"Shop offers and rotations (5.5)":             ShopOffers,
	"Store products and crystal packs (5.6)":      StoreProducts,
	"Pass seasons (5.7)":                          PassSeasons,
	"Missions (5.8)":                              Missions,
	"Medals (5.9)":                                Medals,
	"Leaderboard emblems (5.10)":                  LeaderboardEmblems,
	"Promo codes (5.11)":                          PromoCodes,
	"Ranked seasons (5.12)":                       RankedSeasons,
	"Live events, banners, push campaigns (5.13)": LiveEvents,
	"Config values: economy, rules, penalties, ranking, effects, flags (5.14)":        GameConfig,
	"Config values: anti-cheat (5.14)":                                                AntiCheatConfig,
	"Config values: retention and privacy (5.14)":                                     PrivacyConfig,
	"Config values: moderation and nickname (5.14)":                                   ModerationConfig,
	"Game strings (5.15)":                                                             GameStrings,
	"Word filter and reserved names (5.16)":                                           WordFilter,
	"Live controls (section 4)":                                                       LiveControls,
	"Config snapshots and change sets (5.17)":                                         Snapshots,
	"Players (section 6)":                                                             Players,
	"Review, report and appeal queues (6.4)":                                          Queues,
	"Rights request records (6.5)":                                                    RightsRequests,
	"Content gate reports (`docs/launch-content-gate.md` 4.4); C is running a report": ContentGateReports,
	"Dashboards and aggregate exports (section 10, A-1)":                              Dashboards,
	"Analytics events with analytics id (D-24)":                                       AnalyticsEvents,
	"Audit log (section 7)":                                                           AuditLog,
	"Admin accounts and roles (5.18)":                                                 AdminAccounts,
}

var actionBySpecPhrase = map[string]Action{
	"cultural review":                    CulturalReview,
	"R and P for an incident switch-off": SwitchOffForIncident,
	"approve":                            ApprovePromo,
	"disable":                            DisablePromo,
	"targeted account list":              EditTargetedAccounts,
	"switch":                             SwitchLiveControl,
	"rollback":                           Rollback,
	"rollback (own domains)":             RollbackOwnDomains,
	"all of section 6":                   ActOnPlayer,
	"S":                                  ActOnQueue,
	"grant `super_admin`":                GrantSuperAdmin,
	"export":                             ExportAudit,
	"push to stores":                     PushStorePrices,
}

var viewScopeBySpecQualifier = map[string]Action{
	"counts":                         ViewCounts,
	"entries on objects it can view": View,
	"same":                           View,
	"same, no player entries":        View,
}

var everyAction = []Action{
	View, Create, Edit, Publish, Retire, Delete, ViewCounts, CulturalReview,
	SwitchOffForIncident, ApprovePromo, DisablePromo, EditTargetedAccounts,
	SwitchLiveControl, Rollback, RollbackOwnDomains, ActOnPlayer, ActOnQueue,
	GrantSuperAdmin, ExportAudit, PushStorePrices,
}

func TestEveryRoleIsAllowedExactlyWhatTheAdminSpecGrants(t *testing.T) {
	spec := readSpecMatrix(t)
	if len(spec) != len(matrix) {
		t.Fatalf("spec lists %d objects, matrix %d", len(spec), len(matrix))
	}
	for object, cells := range spec {
		for _, role := range specColumns {
			granted := cells[role]
			for _, action := range everyAction {
				if got := Allows([]Role{role}, object, action); got != granted[action] {
					t.Errorf("%s %s %s: allowed %v, spec %v", role, object, action, got, granted[action])
				}
			}
		}
	}
}

func TestAnAccountWithoutRolesCanDoNothing(t *testing.T) {
	for object := range matrix {
		for _, action := range everyAction {
			if Allows(nil, object, action) {
				t.Errorf("no roles allowed %s %s", object, action)
			}
			if Allows([]Role{"owner"}, object, action) {
				t.Errorf("unknown role allowed %s %s", object, action)
			}
		}
	}
}

func TestPermissionsOfSeveralRolesAddUp(t *testing.T) {
	tests := []struct {
		roles  []Role
		object Object
		action Action
		want   bool
	}{
		{[]Role{Content}, Players, ActOnPlayer, false},
		{[]Role{Content, Support}, Players, ActOnPlayer, true},
		{[]Role{LiveOps}, Medals, Publish, false},
		{[]Role{LiveOps, Content}, Medals, Publish, true},
		{[]Role{Analyst}, PromoCodes, View, false},
		{[]Role{Analyst, Support}, PromoCodes, View, true},
		{[]Role{Support, Analyst}, AdminAccounts, View, false},
	}
	for _, test := range tests {
		if got := Allows(test.roles, test.object, test.action); got != test.want {
			t.Errorf("%v %s %s: allowed %v, want %v", test.roles, test.object, test.action, got, test.want)
		}
	}
}

func readSpecMatrix(t *testing.T) map[Object]map[Role]map[Action]bool {
	t.Helper()
	raw, err := os.ReadFile(adminSpec)
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(raw), "### 2.3 Permission matrix")
	if !found {
		t.Fatal("section 2.3 not found in the admin spec")
	}
	spec := map[Object]map[Role]map[Action]bool{}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "## ") {
			break
		}
		cells := tableCells(line)
		if len(cells) != len(specColumns)+1 || cells[0] == "Object (section)" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		object, known := objectBySpecRow[cells[0]]
		if !known {
			t.Fatalf("spec row %q has no object", cells[0])
		}
		row := map[Role]map[Action]bool{}
		for column, role := range specColumns {
			row[role] = parseSpecCell(t, cells[column+1])
		}
		spec[object] = row
	}
	return spec
}

func tableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil
	}
	parts := strings.Split(trimmed[1:len(trimmed)-1], "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func parseSpecCell(t *testing.T, cell string) map[Action]bool {
	t.Helper()
	granted := map[Action]bool{}
	letters := cell
	if end := strings.IndexFunc(cell, func(r rune) bool { _, ok := actionByLetter[r]; return !ok }); end >= 0 {
		letters, cell = cell[:end], cell[end:]
	} else {
		cell = ""
	}
	for _, letter := range letters {
		granted[actionByLetter[letter]] = true
	}
	if qualifier, ok := strings.CutPrefix(cell, " ("); ok {
		qualifier = strings.TrimSuffix(qualifier, ")")
		scope, known := viewScopeBySpecQualifier[qualifier]
		if !known {
			t.Fatalf("unknown view qualifier %q", qualifier)
		}
		delete(granted, View)
		granted[scope] = true
		return granted
	}
	for _, phrase := range sensitivePhrases(strings.TrimLeft(cell, ",; ")) {
		if action, isLetter := actionByLetter[rune(phrase[0])]; isLetter && len(phrase) == 1 {
			granted[action] = true
			continue
		}
		action, known := actionBySpecPhrase[phrase]
		if !known {
			t.Fatalf("unknown spec phrase %q", phrase)
		}
		granted[action] = true
	}
	return granted
}

func sensitivePhrases(rest string) []string {
	if rest == "" {
		return nil
	}
	if list, ok := strings.CutPrefix(rest, "S: "); ok {
		return strings.Split(list, ", ")
	}
	return []string{rest}
}

func TestObjectsAndRolesAreKnownByName(t *testing.T) {
	objects := Objects()
	if len(objects) != len(matrix) || !slices.IsSorted(objects) {
		t.Fatalf("objects %v", objects)
	}
	for _, object := range objects {
		if !IsObject(object) {
			t.Errorf("%s is not an object", object)
		}
	}
	if IsObject("shop_offer") {
		t.Error("a misspelt object is known")
	}
	for _, role := range specColumns {
		if !IsRole(role) {
			t.Errorf("%s is not a role", role)
		}
	}
	if IsRole("owner") || IsRole("") {
		t.Error("an unknown role is known")
	}
}

func TestGrantedListsExactlyTheAllowedActionsOnce(t *testing.T) {
	roleSets := [][]Role{nil, {Content}, {LiveOps, Content}, {SuperAdmin, Support, Analyst}, {"owner"}}
	for _, roles := range roleSets {
		for object := range matrix {
			granted := Granted(roles, object)
			for _, action := range everyAction {
				if count := countOf(granted, action); count > 1 || (count == 1) != Allows(roles, object, action) {
					t.Errorf("%v %s %s: listed %d times, allowed %v", roles, object, action, count, Allows(roles, object, action))
				}
			}
		}
	}
	if got := Granted([]Role{Support, Analyst}, PromoCodes); !slices.Equal(got, []Action{View, EditTargetedAccounts, ViewCounts}) {
		t.Errorf("support and analyst on promo codes: %v", got)
	}
}

func countOf(actions []Action, action Action) int {
	count := 0
	for _, granted := range actions {
		if granted == action {
			count++
		}
	}
	return count
}
