# Push notifications

Scope: device tokens, notification categories and the server's send path (#132). Campaign definitions and scheduling live in the admin panel (`docs/admin.md` 5.13, #146). Personal data rows: `docs/privacy.md` D-11 and D-22.

## 1. Categories

| Category | Covers | Default | Legal basis |
|---|---|---|---|
| `gameplay` | Rewards ready, friend invites, season reminders, SEV1 notices (`docs/launch-runbook.md`) | On | Contract (D-22) |
| `promotional` | Promotional and event notifications | Off | `promotional_push` consent (`docs/privacy.md` 6.2) |

- **PN-1** A notification is sent only when the account's setting for its category is on. A category that is off sends nothing to any device.
- **PN-2** The OS permission is asked at the first contextual moment, never at first launch (`docs/privacy.md` 6.1). Without it the client does not register a token.
- **PN-3** Turning `promotional` on is the `promotional_push` consent; the client shows it with the other consent purposes and in Settings, Privacy (#157).
- **PN-9** Every change of `promotional` writes one `consent_records` row (`docs/privacy.md` 6.3) in the same transaction as the setting: purpose `promotional_push`, the new state, the policy and app versions from the request, source `settings` and server time. Saving the same value again writes nothing.

## 2. Devices

- **PN-4** A device registers its FCM (Android) or APNs (iOS) token after sign-in and whenever the OS rotates it. A token belongs to one account; registering it again under another account moves it.
- **PN-5** The client removes its token before logout. Logout from every device (`POST /auth/logout-all`) and an account deletion request delete all of the account's tokens on the server. A token that FCM or APNs reports as invalid is deleted at once (D-22 retention).
- **PN-6** Payloads carry game string keys (`titleKey`, `bodyKey`), never resolved text; the device shows them in the player's language (`docs/localization.md`).
- **PN-10** The client keeps the token in memory only. `Game/Push/PushDevice` sends `PUT /push/devices` for a token the server has not yet accepted, one request at a time; a token rotated while a request is in flight is sent after it, and a failed registration is retried on the next `Register` (sign-in or rotation). `Unregister` sends `DELETE /push/devices` and finishes whatever the server answers, so logout is never blocked.

## 3. API

Player HTTP API with the access token (`server/internal/push`):

| Route | Effect |
|---|---|
| `PUT /push/devices` `{token, platform}` | Registers the token; `platform` is `android` or `ios` |
| `DELETE /push/devices` `{token}` | Removes the player's own token |
| `GET /push/settings` | `{gameplay, promotional}` |
| `PUT /push/settings` `{gameplay, promotional, policyVersion, appVersion}` | Changes the categories given, keeps the others, returns `{gameplay, promotional}`. A change of `promotional` needs `policyVersion` (the `PrivacyPolicyVersion` the player saw) and `appVersion`, and writes a consent record (PN-9) |

Errors: `push.token_required`, `push.unknown_platform`, `push.malformed`, `push.consent_context_required` (400).

## 4. Sending

- **PN-7** The server sends through one sender per platform behind a single interface; tests use a fake. FCM and APNs credentials come only from the environment (`docs/environments.md`).
- **PN-8** Owner setup before real sends: a Firebase project with an FCM service account for the Android app, and an APNs authentication key (key id, team id, bundle id) for the iOS app.
