package integrity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth"
)

const (
	DefaultPlayIntegrityEndpoint = "https://playintegrity.googleapis.com"
	playIntegrityScope           = "https://www.googleapis.com/auth/playintegrity"
	playDecodeMaxBody            = 64 * 1024
)

var ErrDecoderUnavailable = errors.New("integrity: attestation service unavailable")

type PlayDecoderConfig struct {
	PackageName        string
	ServiceAccountFile string
	Endpoint           string
	Client             *http.Client
	Now                func() time.Time
}

type PlayDecoder struct {
	target string
	client *http.Client
	source *googleauth.Source
}

func NewPlayDecoder(config PlayDecoderConfig) (*PlayDecoder, error) {
	if config.PackageName == "" || config.ServiceAccountFile == "" {
		return nil, errors.New("integrity: package name and service account are required")
	}
	if config.Endpoint == "" {
		config.Endpoint = DefaultPlayIntegrityEndpoint
	}
	if config.Client == nil {
		config.Client = &http.Client{Timeout: 10 * time.Second}
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	source, err := googleauth.Load(config.ServiceAccountFile, playIntegrityScope, config.Client, config.Now)
	if err != nil {
		return nil, err
	}
	target := config.Endpoint + "/v1/" + url.PathEscape(config.PackageName) + ":decodeIntegrityToken"
	return &PlayDecoder{target: target, client: config.Client, source: source}, nil
}

func (d *PlayDecoder) Decode(ctx context.Context, token string) (PlayIntegrityPayload, error) {
	status, body, err := d.call(ctx, token)
	if err != nil {
		return PlayIntegrityPayload{}, err
	}
	switch {
	case status == http.StatusBadRequest:
		return PlayIntegrityPayload{}, ErrTokenRejected
	case status != http.StatusOK:
		return PlayIntegrityPayload{}, fmt.Errorf("%w: answered %d", ErrDecoderUnavailable, status)
	}
	var decoded struct {
		Payload *PlayIntegrityPayload `json:"tokenPayloadExternal"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.Payload == nil {
		return PlayIntegrityPayload{}, fmt.Errorf("%w: answered no payload", ErrDecoderUnavailable)
	}
	return *decoded.Payload, nil
}

func (d *PlayDecoder) call(ctx context.Context, token string) (int, []byte, error) {
	access, err := d.source.Token(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrDecoderUnavailable, err)
	}
	payload, err := json.Marshal(map[string]string{"integrity_token": token})
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.target, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+access)
	request.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrDecoderUnavailable, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, playDecodeMaxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrDecoderUnavailable, err)
	}
	return response.StatusCode, body, nil
}
