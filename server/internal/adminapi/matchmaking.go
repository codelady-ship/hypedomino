package adminapi

import (
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func matchmakingDomain() configDomain {
	return configDomain{name: matchmaking.Domain, object: rbac.GameConfig, accept: func(_, after []byte) error {
		_, err := matchmaking.Parse(after)
		return err
	}}
}
