package storesync

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

type recordingSyncer struct {
	seen  []storeprice.Target
	write bool
}

func (r *recordingSyncer) Sync(_ context.Context, targets []storeprice.Target, write bool) []Row {
	r.seen, r.write = targets, write
	rows := rowsFor(targets, StatusChange, "")
	if write {
		settle(rows, nil)
	}
	return rows
}

func TestAStoreWithoutCredentialsIsReportedAsNotConfigured(t *testing.T) {
	reports := Stores{}.Run(context.Background(), playTargets, true)
	if len(reports) != 2 || reports[0].Store != GooglePlay || reports[1].Store != AppStore {
		t.Fatalf("reports %+v", reports)
	}
	for _, report := range reports {
		if report.Configured || len(report.Rows) != 0 || len(report.Counts) != 0 {
			t.Fatalf("not configured %+v", report)
		}
	}
}

func TestRunCountsEachStatusAndNeverPassesRussia(t *testing.T) {
	play := &recordingSyncer{}
	targets := append([]storeprice.Target{target("crystals_tier1", "RU", 59)}, playTargets...)
	reports := Stores{GooglePlay: play}.Run(context.Background(), targets, true)
	if !play.write || len(play.seen) != len(playTargets) {
		t.Fatalf("passed %v %v", play.write, play.seen)
	}
	for _, seen := range play.seen {
		if seen.Country == "RU" {
			t.Fatal("RU reached a store")
		}
	}
	if !reports[0].Configured || reports[0].Counts[StatusWritten] != len(playTargets) || reports[1].Configured {
		t.Fatalf("reports %+v", reports)
	}
}

func TestEveryStoreCountryHasItsOwnAlpha3Territory(t *testing.T) {
	alpha3 := regexp.MustCompile(`^[A-Z]{3}$`)
	seen := map[string]string{}
	for _, country := range append(storeprice.StoreCountries(), "RU") {
		territory, known := Territory(country)
		if !known || !alpha3.MatchString(territory) {
			t.Fatalf("%s maps to %q", country, territory)
		}
		if other, taken := seen[territory]; taken {
			t.Fatalf("%s and %s both map to %s", other, country, territory)
		}
		seen[territory] = country
	}
	if len(territories) != len(seen) {
		t.Fatalf("%d mappings for %d countries", len(territories), len(seen))
	}
}

func TestEveryDefaultCountryMapsToItsISOAlpha3Code(t *testing.T) {
	want := strings.Fields(`PT PRT GR GRC CY CYP MT MLT SI SVN CZ CZE SK SVK EE EST LV LVA LT LTU PL POL HU HUN HR HRV CL CHL UY URY PA PAN BN BRN
		AZ AZE TR TUR GE GEO AM ARM KZ KAZ UZ UZB KG KGZ TJ TJK TM TKM BY BLR UA UKR MD MDA RO ROU BG BGR RS SRB BA BIH ME MNE MK MKD AL ALB
		MX MEX BR BRA AR ARG CO COL PE PER EC ECU CR CRI DO DOM MY MYS TH THA CN CHN ZA ZAF MA MAR TN TUN JO JOR LB LBN IQ IRQ MN MNG
		IN IND PK PAK BD BGD LK LKA NP NPL ID IDN PH PHL VN VNM KH KHM LA LAO MM MMR EG EGY NG NGA KE KEN GH GHA TZ TZA UG UGA ET ETH DZ DZA
		BO BOL PY PRY VE VEN HN HND NI NIC GT GTM SV SLV US USA GB GBR DE DEU`)
	expected := map[string]string{}
	for i := 0; i < len(want); i += 2 {
		expected[want[i]] = want[i+1]
	}
	for country := range storeprice.Default().Countries {
		if _, listed := expected[country]; !listed {
			t.Errorf("default country %s has no expected territory here", country)
		}
	}
	for country, territory := range expected {
		if got, _ := Territory(country); got != territory {
			t.Errorf("%s: %s, want %s", country, got, territory)
		}
	}
}
