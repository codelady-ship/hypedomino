package contentpublish

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	ImmutableCacheControl = "public, max-age=31536000, immutable"
	ShortCacheControl     = "public, max-age=300"
	keyRoot               = "addressables"
	bundleType            = "application/octet-stream"
)

var (
	ErrLayout        = errors.New("contentpublish: the build folder does not match the content layout")
	ErrEnvironment   = errors.New("contentpublish: wrong environment")
	ErrAlreadyLive   = errors.New("contentpublish: this catalog version is already published")
	environments     = []string{"dev", "staging", "production"}
	platforms        = []string{"Android", "iOS"}
	versionPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	hashedBundleName = regexp.MustCompile(`^[A-Za-z0-9._-]+_[0-9a-f]{32}\.bundle$`)
)

type Store interface {
	PutObject(ctx context.Context, key string, object objectstore.Object, now time.Time) error
	Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error)
}

type Target struct {
	Environment    string
	Platform       string
	CatalogVersion string
	StoreURL       string
}

type Upload struct {
	File         string
	Key          string
	ContentType  string
	CacheControl string
}

func Key(platform, file string) string {
	return keyRoot + "/" + platform + "/" + file
}

func CatalogFile(version string) string {
	return "catalog_" + version + ".bin"
}

func (t Target) Validate() error {
	if !contains(environments, t.Environment) {
		return fmt.Errorf("%w: %q is not one of %s", ErrEnvironment, t.Environment, strings.Join(environments, ", "))
	}
	if !contains(platforms, t.Platform) {
		return fmt.Errorf("%w: platform %q is not one of %s", ErrLayout, t.Platform, strings.Join(platforms, ", "))
	}
	if !versionPattern.MatchString(t.CatalogVersion) {
		return fmt.Errorf("%w: catalog version %q", ErrLayout, t.CatalogVersion)
	}
	return CheckStoreURL(t.Environment, t.StoreURL)
}

func CheckStoreURL(environment, storeURL string) error {
	url := strings.ToLower(storeURL)
	if !strings.Contains(url, environment) {
		return fmt.Errorf("%w: the store url must name the %s environment", ErrEnvironment, environment)
	}
	for _, other := range environments {
		if other != environment && strings.Contains(url, other) {
			return fmt.Errorf("%w: the store url names %s, not %s", ErrEnvironment, other, environment)
		}
	}
	return nil
}

func Plan(dir string, target Target) ([]Upload, error) {
	if err := target.Validate(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	catalog := CatalogFile(target.CatalogVersion)
	var bundles []Upload
	var catalogUpload *Upload
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("%w: unexpected folder %s", ErrLayout, entry.Name())
		}
		upload, err := classify(entry.Name(), target.Platform, catalog)
		if err != nil {
			return nil, err
		}
		if entry.Name() == catalog {
			catalogUpload = &upload
			continue
		}
		bundles = append(bundles, upload)
	}
	if catalogUpload == nil {
		return nil, fmt.Errorf("%w: %s is missing", ErrLayout, catalog)
	}
	sort.Slice(bundles, func(a, b int) bool { return bundles[a].File < bundles[b].File })
	return append(bundles, *catalogUpload), nil
}

func classify(file, platform, catalog string) (Upload, error) {
	upload := Upload{File: file, Key: Key(platform, file), ContentType: bundleType, CacheControl: ShortCacheControl}
	switch {
	case strings.HasSuffix(file, ".bundle"):
		if !hashedBundleName.MatchString(file) {
			return Upload{}, fmt.Errorf("%w: bundle %s has no content hash in its name (CD-3)", ErrLayout, file)
		}
		upload.CacheControl = ImmutableCacheControl
	case file == catalog:
	case strings.HasPrefix(file, "catalog_") && strings.TrimSuffix(file, filepath.Ext(file)) != strings.TrimSuffix(catalog, ".bin"):
		return Upload{}, fmt.Errorf("%w: %s belongs to another catalog version", ErrLayout, file)
	case strings.HasSuffix(file, ".json"), strings.HasSuffix(file, ".hash"):
		upload.ContentType = "application/json"
	default:
		return Upload{}, fmt.Errorf("%w: unexpected file %s", ErrLayout, file)
	}
	return upload, nil
}

func Publish(ctx context.Context, store Store, dir string, uploads []Upload, now time.Time) (int, error) {
	last := uploads[len(uploads)-1]
	live, err := exists(ctx, store, last.Key, now)
	if err != nil {
		return 0, err
	}
	if live {
		return 0, fmt.Errorf("%w: %s", ErrAlreadyLive, last.Key)
	}
	sent := 0
	for _, upload := range uploads {
		if upload.CacheControl == ImmutableCacheControl {
			present, err := exists(ctx, store, upload.Key, now)
			if err != nil {
				return sent, err
			}
			if present {
				continue
			}
		}
		body, err := os.ReadFile(filepath.Join(dir, upload.File))
		if err != nil {
			return sent, err
		}
		object := objectstore.Object{ContentType: upload.ContentType, CacheControl: upload.CacheControl, Body: body}
		if err := store.PutObject(ctx, upload.Key, object, now); err != nil {
			return sent, fmt.Errorf("%s: %w", upload.Key, err)
		}
		sent++
	}
	return sent, nil
}

func exists(ctx context.Context, store Store, key string, now time.Time) (bool, error) {
	body, err := store.Open(ctx, key, now)
	if errors.Is(err, objectstore.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	body.Close()
	return true, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
