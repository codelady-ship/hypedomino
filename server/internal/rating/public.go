package rating

import "context"

type PublicRanked struct {
	Ranks   map[string]string `json:"ranks"`
	Emblems []Emblem          `json:"emblems"`
}

func (s *Store) Public(ctx context.Context, account string) (PublicRanked, error) {
	public := PublicRanked{Ranks: map[string]string{}}
	for _, mode := range modes {
		label, err := s.Label(ctx, account, mode)
		if err != nil {
			return PublicRanked{}, err
		}
		if label != "" {
			public.Ranks[mode.String()] = label
		}
	}
	emblems, err := s.showcaseEmblems(ctx, account)
	public.Emblems = emblems
	return public, err
}
