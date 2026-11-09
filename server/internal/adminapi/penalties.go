package adminapi

import (
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func penaltiesDomain() configDomain {
	return configDomain{name: penalties.Domain, object: rbac.GameConfig, accept: func(_, after []byte) error {
		_, err := penalties.Parse(after)
		return err
	}}
}
