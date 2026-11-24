package objectstore

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	amzTime  = "20060102T150405Z"
	amzDay   = "20060102"
	maxTTL   = 7 * 24 * time.Hour
	service  = "s3"
	scopeEnd = "aws4_request"
)

var (
	ErrNotFound    = errors.New("objectstore: no such object")
	ErrUnavailable = errors.New("objectstore: the store did not answer")
	ErrConfig      = errors.New("objectstore: invalid config")
)

type Config struct {
	BaseURL   string
	Region    string
	AccessKey string
	SecretKey string
	URLTTL    time.Duration
}

type Store struct {
	base   *url.URL
	config Config
	client *http.Client
}

func New(config Config) (*Store, error) {
	base, err := url.Parse(strings.TrimRight(config.BaseURL, "/"))
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, fmt.Errorf("%w: base url %q", ErrConfig, config.BaseURL)
	}
	if config.Region == "" || config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("%w: region, access key and secret key are required", ErrConfig)
	}
	if config.URLTTL <= 0 || config.URLTTL > maxTTL {
		return nil, fmt.Errorf("%w: url lifetime %s is outside 0 to %s", ErrConfig, config.URLTTL, maxTTL)
	}
	return &Store{base: base, config: config, client: &http.Client{}}, nil
}

func (s *Store) SignedPut(key string, now time.Time) string {
	return s.signed(http.MethodPut, key, now)
}

func (s *Store) SignedGet(key string, now time.Time) string {
	return s.signed(http.MethodGet, key, now)
}

func (s *Store) Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.SignedGet(key, now), nil)
	if err != nil {
		return nil, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	switch response.StatusCode {
	case http.StatusOK:
		return response.Body, nil
	case http.StatusNotFound:
		response.Body.Close()
		return nil, ErrNotFound
	}
	response.Body.Close()
	return nil, fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
}

type Object struct {
	ContentType  string
	CacheControl string
	Body         []byte
}

func (s *Store) Put(ctx context.Context, key, contentType string, body []byte, now time.Time) error {
	return s.PutObject(ctx, key, Object{ContentType: contentType, Body: body}, now)
}

func (s *Store) PutObject(ctx context.Context, key string, object Object, now time.Time) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, s.SignedPut(key, now), bytes.NewReader(object.Body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", object.ContentType)
	if object.CacheControl != "" {
		request.Header.Set("Cache-Control", object.CacheControl)
	}
	return s.send(request)
}

func (s *Store) Delete(ctx context.Context, key string, now time.Time) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.signed(http.MethodDelete, key, now), nil)
	if err != nil {
		return err
	}
	return s.send(request)
}

func (s *Store) send(request *http.Request) error {
	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	response.Body.Close()
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}
	return nil
}

type Entry struct {
	Key          string
	LastModified time.Time
}

type listPage struct {
	Contents []struct {
		Key          string
		LastModified time.Time
	}
	IsTruncated           bool
	NextContinuationToken string
}

func (s *Store) List(ctx context.Context, prefix string, now time.Time) ([]Entry, error) {
	var entries []Entry
	token := ""
	for {
		page, err := s.listPage(ctx, prefix, token, now)
		if err != nil {
			return nil, err
		}
		for _, content := range page.Contents {
			entries = append(entries, Entry{Key: content.Key, LastModified: content.LastModified})
		}
		if !page.IsTruncated {
			return entries, nil
		}
		if page.NextContinuationToken == "" {
			return nil, fmt.Errorf("%w: a truncated listing has no continuation token", ErrUnavailable)
		}
		token = page.NextContinuationToken
	}
}

func (s *Store) listPage(ctx context.Context, prefix, token string, now time.Time) (listPage, error) {
	query := url.Values{"list-type": {"2"}, "prefix": {prefix}}
	if token != "" {
		query.Set("continuation-token", token)
	}
	path := s.base.EscapedPath()
	if path == "" {
		path = "/"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.sign(http.MethodGet, path, query, now), nil)
	if err != nil {
		return listPage{}, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return listPage{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return listPage{}, fmt.Errorf("%w: list status %d", ErrUnavailable, response.StatusCode)
	}
	var page listPage
	if err := xml.NewDecoder(response.Body).Decode(&page); err != nil {
		return listPage{}, fmt.Errorf("%w: list response: %v", ErrUnavailable, err)
	}
	return page, nil
}

func (s *Store) signed(method, key string, now time.Time) string {
	return s.sign(method, s.base.EscapedPath()+"/"+escapePath(key), url.Values{}, now)
}

func (s *Store) sign(method, path string, query url.Values, now time.Time) string {
	now = now.UTC()
	day := now.Format(amzDay)
	scope := strings.Join([]string{day, s.config.Region, service, scopeEnd}, "/")
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", s.config.AccessKey+"/"+scope)
	query.Set("X-Amz-Date", now.Format(amzTime))
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(s.config.URLTTL/time.Second), 10))
	query.Set("X-Amz-SignedHeaders", "host")
	encoded := strings.ReplaceAll(query.Encode(), "+", "%20")
	canonical := strings.Join([]string{method, path, encoded, "host:" + s.base.Host, "", "host", "UNSIGNED-PAYLOAD"}, "\n")
	toSign := strings.Join([]string{"AWS4-HMAC-SHA256", now.Format(amzTime), scope, hexHash(canonical)}, "\n")
	signature := hex.EncodeToString(mac(s.signingKey(day), toSign))
	return s.base.Scheme + "://" + s.base.Host + path + "?" + encoded + "&X-Amz-Signature=" + signature
}

func (s *Store) signingKey(day string) []byte {
	key := mac([]byte("AWS4"+s.config.SecretKey), day)
	key = mac(key, s.config.Region)
	key = mac(key, service)
	return mac(key, scopeEnd)
}

func escapePath(key string) string {
	segments := strings.Split(key, "/")
	for at, segment := range segments {
		segments[at] = strings.ReplaceAll(url.QueryEscape(segment), "+", "%20")
	}
	return strings.Join(segments, "/")
}

func mac(key []byte, message string) []byte {
	digest := hmac.New(sha256.New, key)
	digest.Write([]byte(message))
	return digest.Sum(nil)
}

func hexHash(message string) string {
	sum := sha256.Sum256([]byte(message))
	return hex.EncodeToString(sum[:])
}
