package contentpublish

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	catalogPrefix    = "catalog_"
	catalogExtension = ".bin"
	maxBundleName    = 255
)

var (
	ErrRetention          = errors.New("contentpublish: invalid retention")
	ErrCatalog            = errors.New("contentpublish: a catalog in scope cannot be read")
	releaseCatalogVersion = regexp.MustCompile(`^((0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5}))-[0-9]+$`)
	bundleNameEnd         = regexp.MustCompile(`_[0-9a-f]{32}\.bundle`)
)

type Bucket interface {
	List(ctx context.Context, prefix string, now time.Time) ([]objectstore.Entry, error)
	Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error)
	Delete(ctx context.Context, key string, now time.Time) error
}

type Retention struct {
	Target        Target
	MinAppVersion string
	Grace         time.Duration
}

type CatalogScope struct {
	Key       string
	Supported bool
	Reason    string
}

type Sweep struct {
	Catalogs     []CatalogScope
	Referenced   []string
	Young        []string
	Unreferenced []string
	Missing      []string
}

func (r Retention) Validate() error {
	if err := r.Target.Validate(); err != nil {
		return err
	}
	if !gateway.VersionAtLeast(r.MinAppVersion, "0.0.0") {
		return fmt.Errorf("%w: minimum app version %q is not major.minor.patch", ErrRetention, r.MinAppVersion)
	}
	if r.Grace <= 0 {
		return fmt.Errorf("%w: the grace period must be positive, got %s", ErrRetention, r.Grace)
	}
	return nil
}

func PlanSweep(ctx context.Context, bucket Bucket, retention Retention, now time.Time) (Sweep, error) {
	if err := retention.Validate(); err != nil {
		return Sweep{}, err
	}
	prefix := Key(retention.Target.Platform, "")
	entries, err := bucket.List(ctx, prefix, now)
	if err != nil {
		return Sweep{}, err
	}
	catalogs, bundles, err := split(entries, prefix)
	if err != nil {
		return Sweep{}, err
	}
	active := Key(retention.Target.Platform, CatalogFile(retention.Target.CatalogVersion))
	if _, ok := catalogs[active]; !ok {
		return Sweep{}, fmt.Errorf("%w: the active catalog %s is not in the store", ErrCatalog, active)
	}
	var sweep Sweep
	referenced := map[string]bool{}
	for _, key := range sortedKeys(catalogs) {
		scope := scopeOf(key, catalogs[key], active, retention.MinAppVersion)
		sweep.Catalogs = append(sweep.Catalogs, scope)
		if !scope.Supported {
			continue
		}
		names, err := catalogBundles(ctx, bucket, key, now)
		if err != nil {
			return Sweep{}, err
		}
		for _, name := range names {
			referenced[Key(retention.Target.Platform, name)] = true
		}
	}
	for _, key := range sortedKeys(bundles) {
		switch {
		case referenced[key]:
			sweep.Referenced = append(sweep.Referenced, key)
		case now.Sub(bundles[key]) < retention.Grace:
			sweep.Young = append(sweep.Young, key)
		default:
			sweep.Unreferenced = append(sweep.Unreferenced, key)
		}
	}
	for _, key := range sortedKeys(referenced) {
		if _, ok := bundles[key]; !ok {
			sweep.Missing = append(sweep.Missing, key)
		}
	}
	return sweep, nil
}

func DeleteUnreferenced(ctx context.Context, bucket Bucket, sweep Sweep, now time.Time) (int, error) {
	for deleted, key := range sweep.Unreferenced {
		if err := bucket.Delete(ctx, key, now); err != nil {
			return deleted, fmt.Errorf("%s: %w", key, err)
		}
	}
	return len(sweep.Unreferenced), nil
}

func split(entries []objectstore.Entry, prefix string) (map[string]string, map[string]time.Time, error) {
	catalogs := map[string]string{}
	bundles := map[string]time.Time{}
	for _, entry := range entries {
		name, ok := strings.CutPrefix(entry.Key, prefix)
		if !ok || strings.Contains(name, "/") {
			continue
		}
		if strings.HasPrefix(name, catalogPrefix) && strings.HasSuffix(name, catalogExtension) {
			version := strings.TrimSuffix(strings.TrimPrefix(name, catalogPrefix), catalogExtension)
			if !versionPattern.MatchString(version) {
				return nil, nil, fmt.Errorf("%w: %s has no valid catalog version", ErrCatalog, entry.Key)
			}
			catalogs[entry.Key] = version
			continue
		}
		if hashedBundleName.MatchString(name) {
			bundles[entry.Key] = entry.LastModified
		}
	}
	return catalogs, bundles, nil
}

func scopeOf(key, version, active, minimum string) CatalogScope {
	if key == active {
		return CatalogScope{Key: key, Supported: true, Reason: "active catalog"}
	}
	match := releaseCatalogVersion.FindStringSubmatch(version)
	if match == nil {
		return CatalogScope{Key: key, Supported: true, Reason: "no app version in the catalog version"}
	}
	if gateway.VersionAtLeast(match[1], minimum) {
		return CatalogScope{Key: key, Supported: true, Reason: "app " + match[1] + " is supported"}
	}
	return CatalogScope{Key: key, Supported: false, Reason: "app " + match[1] + " is below " + minimum}
}

func catalogBundles(ctx context.Context, bucket Bucket, key string, now time.Time) ([]string, error) {
	body, err := bucket.Open(ctx, key, now)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCatalog, key, err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCatalog, key, err)
	}
	names := BundlesInCatalog(data)
	if len(names) == 0 {
		return nil, fmt.Errorf("%w: %s names no content-hashed bundle", ErrCatalog, key)
	}
	return names, nil
}

func BundlesInCatalog(catalog []byte) []string {
	found := map[string]bool{}
	for _, end := range bundleNameEnd.FindAllIndex(catalog, -1) {
		if name, ok := lengthPrefixedName(catalog, end[1]); ok {
			found[name] = true
		}
	}
	return sortedKeys(found)
}

func lengthPrefixedName(catalog []byte, end int) (string, bool) {
	for start := end - 1; start >= 4 && end-start <= maxBundleName; start-- {
		if binary.LittleEndian.Uint32(catalog[start-4:start]) != uint32(end-start) {
			continue
		}
		if hashedBundleName.Match(catalog[start:end]) {
			return string(catalog[start:end]), true
		}
	}
	return "", false
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
