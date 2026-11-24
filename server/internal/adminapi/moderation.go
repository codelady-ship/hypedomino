package adminapi

import (
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func moderationDomain() configDomain {
	return configDomain{name: profile.ModerationDomain, object: rbac.ModerationConfig, accept: func(_, after []byte) error {
		_, err := profile.ParseModeration(after)
		return err
	}}
}
