package gamestrings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

const Domain = "localization"

var ErrInvalidLocalization = errors.New("gamestrings: invalid localization snapshot")

type Localization struct {
	Bundles        map[Language]int        `json:"bundles"`
	FallbackChains map[Language][]Language `json:"fallbackChains"`
}

func DefaultFallbackChains() map[Language][]Language {
	return map[Language][]Language{
		Azerbaijani: {Azerbaijani, Turkish, English},
		English:     {English},
		Turkish:     {Turkish, English},
		Russian:     {Russian, English},
	}
}

func ParseLocalization(raw []byte) (Localization, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var localization Localization
	if err := decoder.Decode(&localization); err != nil {
		return Localization{}, fmt.Errorf("%w: %w", ErrInvalidLocalization, err)
	}
	if err := checkBundles(localization.Bundles); err != nil {
		return Localization{}, err
	}
	chains := DefaultFallbackChains()
	for language, chain := range localization.FallbackChains {
		if err := checkChain(language, chain); err != nil {
			return Localization{}, err
		}
		chains[language] = chain
	}
	localization.FallbackChains = chains
	return localization, nil
}

func checkBundles(bundles map[Language]int) error {
	if _, has := bundles[English]; !has {
		return fmt.Errorf("%w: no en bundle", ErrInvalidLocalization)
	}
	for language, version := range bundles {
		if !slices.Contains(Languages, language) {
			return fmt.Errorf("%w: unknown language %q", ErrInvalidLocalization, language)
		}
		if version <= 0 {
			return fmt.Errorf("%w: %s bundle version %d", ErrInvalidLocalization, language, version)
		}
	}
	return nil
}

func checkChain(language Language, chain []Language) error {
	switch {
	case !slices.Contains(Languages, language):
		return fmt.Errorf("%w: chain for unknown language %q", ErrInvalidLocalization, language)
	case len(chain) == 0 || chain[0] != language:
		return fmt.Errorf("%w: %s chain must start with %s", ErrInvalidLocalization, language, language)
	case chain[len(chain)-1] != English:
		return fmt.Errorf("%w: %s chain must end with en", ErrInvalidLocalization, language)
	}
	for i, step := range chain {
		if !slices.Contains(Languages, step) || slices.Contains(chain[:i], step) {
			return fmt.Errorf("%w: %s chain %v", ErrInvalidLocalization, language, chain)
		}
	}
	return nil
}

type ActiveBundle struct {
	Language Language
	Version  int
	Hash     string
}

type Active struct {
	Bundles        []ActiveBundle
	FallbackChains map[Language][]Language
}

func (s *Store) Resolve(ctx context.Context, localization Localization) (Active, error) {
	active := Active{FallbackChains: localization.FallbackChains}
	var english Published
	for _, language := range Languages {
		version, named := localization.Bundles[language]
		if !named {
			continue
		}
		bundle, err := s.Bundle(ctx, language, version)
		if err != nil {
			return Active{}, fmt.Errorf("%w: %s version %d: %w", ErrInvalidLocalization, language, version, err)
		}
		if language == English {
			english = bundle
		}
		active.Bundles = append(active.Bundles, ActiveBundle{Language: language, Version: version, Hash: bundle.Hash})
	}
	if err := s.checkComplete(ctx, english); err != nil {
		return Active{}, err
	}
	return active, nil
}

func (s *Store) checkComplete(ctx context.Context, english Published) error {
	if english.Version == 0 {
		return fmt.Errorf("%w: no en bundle", ErrInvalidLocalization)
	}
	var bundle Bundle
	if err := json.Unmarshal(english.Content, &bundle); err != nil {
		return err
	}
	entries, err := s.Entries(ctx)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, has := bundle.Strings[entry.Key]; !entry.Deprecated && !has {
			return fmt.Errorf("%w: en version %d lacks %s", ErrInvalidLocalization, english.Version, entry.Key)
		}
	}
	return nil
}

func (s *Store) ActiveParser(ctx context.Context) func(raw []byte) (Active, error) {
	return func(raw []byte) (Active, error) {
		localization, err := ParseLocalization(raw)
		if err != nil {
			return Active{}, err
		}
		return s.Resolve(ctx, localization)
	}
}

func (a Active) Hello() ([]*pb.StringBundle, []*pb.FallbackChain) {
	bundles := make([]*pb.StringBundle, 0, len(a.Bundles))
	for _, bundle := range a.Bundles {
		bundles = append(bundles, &pb.StringBundle{Language: string(bundle.Language), Version: uint32(bundle.Version), Hash: bundle.Hash})
	}
	chains := make([]*pb.FallbackChain, 0, len(a.FallbackChains))
	for _, language := range Languages {
		chain, has := a.FallbackChains[language]
		if !has {
			continue
		}
		steps := make([]string, len(chain))
		for i, step := range chain {
			steps[i] = string(step)
		}
		chains = append(chains, &pb.FallbackChain{Language: string(language), Languages: steps})
	}
	return bundles, chains
}
