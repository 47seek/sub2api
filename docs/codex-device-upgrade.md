# Codex Device Profile: 0.2.1 to 0.2.15

This branch targets a single sub2api instance, ChatGPT OAuth accounts, ordinary
HTTP forwarding, and downstream Codex clients using a sub2api API key with a
custom provider. It is based on official sub2api 0.2.15. It adds no database
schema or migration. Passthrough and WebSocket workflows are outside this
configuration's acceptance scope.

## Account and Version Settings

Use the existing account editor to select `device` for every active OpenAI OAuth
account. This stores `extra.codex_fingerprint_mode="device"`. Disable
`extra.openai_passthrough` and the legacy `extra.openai_oauth_passthrough` flag.
Preserve each existing account row and its system-managed
`extra.codex_fingerprint_seed`; do not replace rows by reimporting accounts or
copy a seed between accounts. The existing save path creates a missing seed.
Keep each account's assigned proxy.

Save all five settings below through the admin UI or the existing settings API.
They are database settings, not environment variables. They can be saved on the
old installation before replacing the image; official migrations do not upgrade
an existing 0.153.4 setting to 0.162.0.

```bash
curl --fail-with-body --silent --show-error \
  -H "Authorization: Bearer ${SUB2API_ADMIN_TOKEN}" \
  -H 'Content-Type: application/json' \
  -X PUT "${SUB2API_URL%/}/api/v1/admin/settings" \
  --data '{
    "openai_codex_client_version": "0.162.0",
    "openai_codex_user_agent": "codex-tui/0.162.0 (Ubuntu 24.4.0; x86_64) xterm-256color (codex-tui; 0.162.0)",
    "openai_codex_version_auto_sync_enabled": false,
    "min_codex_version": "0.162.0",
    "max_codex_version": ""
  }'
```

The API invalidates the relevant local setting caches; these settings do not
require a restart. The explicit canonical version also controls the device-mode
upstream models query. A client below 0.162.0 is rejected. The maximum is empty,
so newer clients are admitted, but the fixed 0.162.0 catalog may lack their newer
capabilities. This is not a promise of compatibility with every future release.
The canonical UA selects an outbound profile; it does not rewrite the client's
actual OS, paths, shell, tools, or permissions.

## Downstream Provider

Keep the existing sub2api API-key authentication method. Set `wire_api` to
`responses` and explicitly disable WebSockets in the selected provider. A new
provider using an environment-held API key can use this configuration:

```toml
model_provider = "sub2api"

[model_providers.sub2api]
name = "OpenAI"
base_url = "https://gateway.example/v1"
wire_api = "responses"
env_key = "SUB2API_API_KEY"
requires_openai_auth = false
supports_websockets = false
model_catalog_url = "https://gateway.example/backend-api/codex/models"
```

Use the gateway's actual address and any deployment prefix. The catalog URL is
a complete URL in the provider table, not `/v1/models`. Remove a top-level
`model_catalog_json` override when selecting this remote catalog, and start a new
Codex process. An existing provider that stores the sub2api key through Codex's
current auth mechanism does not need to switch to `env_key` solely for this
upgrade. Keep real local execution settings unchanged.

## Direct Image Upgrade

Direct replacement is supported when the old database uses the official 0.2.1
migration history and checksums, with no incompatible manual schema changes.
Preserve the database, Redis and application data volumes, account IDs and seeds,
API keys, group associations, `JWT_SECRET`, and `TOTP_ENCRYPTION_KEY`. Take a
consistent pre-upgrade database and application-data backup. This branch does
not need a Redis flush: auth snapshot version 25 rejects both old custom and
official version 24 formats and rebuilds them from the database.

Review every enabled `models_list_config` before upgrading. Official migrations
rename it to `model_allowlist`, and its list now restricts actual requests.
Models outside that list return 404, even if the previous version only used the
list for display. Account model mappings still apply.

Load the delivered local image into the deployment's Docker daemon. From the
existing deployment directory, add the supplied override last so it replaces
any older custom-image override:

```bash
export CODEX_DEVICE_IMAGE='sub2api:0.2.15-codex-device.1'
export CODEX_DEVICE_OVERRIDE='/path/to/checkout/deploy/docker-compose.codex-device.yml'

docker compose --env-file .env -f compose.yaml -f compose.codex-device.yaml \
  -f "$CODEX_DEVICE_OVERRIDE" config --images
docker compose --env-file .env -f compose.yaml -f compose.codex-device.yaml \
  -f "$CODEX_DEVICE_OVERRIDE" up -d --no-deps --force-recreate sub2api
docker compose --env-file .env -f compose.yaml -f compose.codex-device.yaml \
  -f "$CODEX_DEVICE_OVERRIDE" logs --tail=100 sub2api
curl --fail --silent --show-error "${SUB2API_URL%/}/health"
```

Verify the loaded image against the delivery manifest before applying it. The
override retains existing mounts, secrets, ports and database/Redis services.
It forces upstream HTTP and enables identity enforcement. `restart` alone does
not apply a new image or changed Compose environment. These commands are
deployment instructions, not evidence that production has been upgraded.

The new application runs 11 official migrations during startup; no intermediate
application version is required. Each migration has its own transaction, not a
single transaction covering the entire upgrade. Startup migrations have a
10-minute overall deadline, so database size and lock contention still matter.
Check startup/health, the five settings, account mode, group allowlists and an
ordinary Codex request after replacement.

Switching only the image back to 0.2.1 is not a reliable rollback: official
migrations rename columns, remove some quota rows and transform pricing JSON.
Rollback would require the consistent pre-upgrade database/data backup and its
matching image/configuration. No rollback rehearsal is part of this delivery.

## Maintained Custom Behavior and Limits

The customization retains native session/thread/turn/window/cache references
and fixes installation identity per local account row. Soft affinity and the
existing bounded account switching remain. Explicit encrypted-content errors
register the rejected request's ciphertext only for the selected row and owner;
cleaning can also remove valid ciphertext in that request. Usable reasoning
text, other messages, tool calls and results remain; invalid compaction items
can be removed. Recovery is bounded.

Turn-state provenance follows each concrete value and is accepted only for the
current row and owner. Unknown or expired provenance is stripped. Cookies use
the native allowlist and expiry in row-specific jars independent of connection
pool eviction. Cookie, provenance and invalid-ciphertext records live only in
this process and disappear on restart; same-owner token refresh does not clear
the row's cookies.

The fixed uTLS profile covers generation, models and existing account-related
HTTP clients while preserving endpoint semantics. Unsupported HTTPS proxies
and transport failures do not silently fall back to another TLS fingerprint.
Use the account's supported HTTP or SOCKS proxy configuration. Updating the
native baseline later requires rechecking this profile and request semantics;
automatic version synchronization must remain disabled for this setup.

The fixed profile is not the complete native transport stack. Real execution
environment differences remain. Request success or matching JA3 values do not
prove that the upstream classifies the traffic as one device. Real cross-owner
behavior, all auxiliary workflows and future clients have not been verified.
Real OAuth refresh was explicitly excluded from verification. The delivery
record distinguishes mock checks, prior real samples and the final image's
isolated upgrade from unexamined production data and configuration.
