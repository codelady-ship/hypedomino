package push_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/push"
)

type fakeRecipients struct {
	settings  push.Settings
	devices   []push.Device
	forgotten []string
}

func (f *fakeRecipients) Settings(context.Context, string) (push.Settings, error) {
	return f.settings, nil
}

func (f *fakeRecipients) Devices(context.Context, string) ([]push.Device, error) {
	return f.devices, nil
}

func (f *fakeRecipients) Forget(_ context.Context, token string) error {
	f.forgotten = append(f.forgotten, token)
	return nil
}

type fakeSender struct {
	sent     []string
	rejected map[string]error
}

func (f *fakeSender) Send(_ context.Context, device push.Device, message push.Message) error {
	if err := f.rejected[device.Token]; err != nil {
		return err
	}
	f.sent = append(f.sent, device.Token+":"+message.TitleKey)
	return nil
}

var (
	phone  = push.Device{Token: "fcm-1", Platform: push.Android}
	tablet = push.Device{Token: "apns-1", Platform: push.IOS}
)

func TestNotifyRespectsTheCategorySetting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings push.Settings
		category push.Category
		want     []string
	}{
		{"gameplay on by default", push.DefaultSettings, push.Gameplay, []string{"fcm-1:push.reward.title", "apns-1:push.reward.title"}},
		{"promotional off by default", push.DefaultSettings, push.Promotional, nil},
		{"gameplay turned off", push.Settings{Promotional: true}, push.Gameplay, nil},
		{"promotional turned on", push.Settings{Promotional: true}, push.Promotional, []string{"fcm-1:push.reward.title", "apns-1:push.reward.title"}},
		{"unknown category", push.Settings{Gameplay: true, Promotional: true}, "marketing", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recipients := &fakeRecipients{settings: tc.settings, devices: []push.Device{phone, tablet}}
			sender := &fakeSender{}
			sent, err := push.NewNotifier(recipients, sender).Notify(context.Background(), "ayla", push.Message{Category: tc.category, TitleKey: "push.reward.title", BodyKey: "push.reward.body"})
			if err != nil || sent != len(tc.want) || !slices.Equal(sender.sent, tc.want) {
				t.Fatalf("sent %d %v %v, want %v", sent, sender.sent, err, tc.want)
			}
		})
	}
}

func TestNotifyForgetsTokensTheProviderRejects(t *testing.T) {
	recipients := &fakeRecipients{settings: push.DefaultSettings, devices: []push.Device{phone, tablet}}
	sender := &fakeSender{rejected: map[string]error{"fcm-1": push.ErrInvalidToken}}
	sent, err := push.NewNotifier(recipients, sender).Notify(context.Background(), "ayla", push.Message{Category: push.Gameplay, TitleKey: "t"})
	if err != nil || sent != 1 || !slices.Equal(recipients.forgotten, []string{"fcm-1"}) {
		t.Fatalf("sent %d forgotten %v %v", sent, recipients.forgotten, err)
	}
}

func TestNotifyKeepsSendingAfterAProviderFailure(t *testing.T) {
	down := errors.New("provider down")
	recipients := &fakeRecipients{settings: push.DefaultSettings, devices: []push.Device{phone, tablet}}
	sender := &fakeSender{rejected: map[string]error{"fcm-1": down}}
	sent, err := push.NewNotifier(recipients, sender).Notify(context.Background(), "ayla", push.Message{Category: push.Gameplay, TitleKey: "t"})
	if !errors.Is(err, down) || sent != 1 || recipients.forgotten != nil {
		t.Fatalf("sent %d forgotten %v %v", sent, recipients.forgotten, err)
	}
}
