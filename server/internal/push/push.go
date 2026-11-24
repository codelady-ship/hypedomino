package push

import (
	"context"
	"errors"
)

type Category string

const (
	Gameplay    Category = "gameplay"
	Promotional Category = "promotional"
)

const (
	Android = "android"
	IOS     = "ios"
)

var (
	ErrNoToken         = errors.New("push.token_required")
	ErrUnknownPlatform = errors.New("push.unknown_platform")
	ErrMalformed       = errors.New("push.malformed")
	ErrConsentContext  = errors.New("push.consent_context_required")
	ErrInvalidToken    = errors.New("push: token rejected by the provider")
)

type Settings struct {
	Gameplay    bool `json:"gameplay"`
	Promotional bool `json:"promotional"`
}

var DefaultSettings = Settings{Gameplay: true}

type SettingsChange struct {
	Settings
	PolicyVersion string `json:"policyVersion"`
	AppVersion    string `json:"appVersion"`
}

func (s Settings) Allows(category Category) bool {
	switch category {
	case Gameplay:
		return s.Gameplay
	case Promotional:
		return s.Promotional
	}
	return false
}

type Device struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (d Device) Validate() error {
	if d.Token == "" {
		return ErrNoToken
	}
	if d.Platform != Android && d.Platform != IOS {
		return ErrUnknownPlatform
	}
	return nil
}

type Message struct {
	Category Category
	TitleKey string
	BodyKey  string
}

type Sender interface {
	Send(ctx context.Context, device Device, message Message) error
}

type Recipients interface {
	Settings(ctx context.Context, account string) (Settings, error)
	Devices(ctx context.Context, account string) ([]Device, error)
	Forget(ctx context.Context, token string) error
}

type Notifier struct {
	recipients Recipients
	sender     Sender
}

func NewNotifier(recipients Recipients, sender Sender) *Notifier {
	return &Notifier{recipients: recipients, sender: sender}
}

func (n *Notifier) Notify(ctx context.Context, account string, message Message) (int, error) {
	settings, err := n.recipients.Settings(ctx, account)
	if err != nil || !settings.Allows(message.Category) {
		return 0, err
	}
	devices, err := n.recipients.Devices(ctx, account)
	if err != nil {
		return 0, err
	}
	sent := 0
	var failures []error
	for _, device := range devices {
		err := n.sender.Send(ctx, device, message)
		switch {
		case errors.Is(err, ErrInvalidToken):
			failures = append(failures, n.recipients.Forget(ctx, device.Token))
		case err != nil:
			failures = append(failures, err)
		default:
			sent++
		}
	}
	return sent, errors.Join(failures...)
}
