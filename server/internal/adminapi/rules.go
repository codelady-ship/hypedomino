package adminapi

import (
	"github.com/Alievs-corp/hypedomino/server/internal/rulesconfig"
)

func rulesDomain() keyedDomain {
	keys := make([]configKey, 0, len(rulesconfig.Keys()))
	for _, key := range rulesconfig.Keys() {
		keys = append(keys, configKey{Name: key.Name, Unit: key.Unit, Min: key.Min, Max: &key.Max, Default: key.Default()})
	}
	return newKeyedDomain(rulesconfig.Domain, keys, func(raw []byte) (map[string]int, error) {
		return rulesconfig.Parse(raw)
	})
}
