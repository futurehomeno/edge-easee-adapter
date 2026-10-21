# Telemetry on by default

## Why
Easee seeds a disabled telemetry block, so telemetry stays off until the cloud enables it. The other
adapters keep cliffhanger's default: enabled for 30 days on a fresh install.

## What Changes
- Drop the disabled seed; a missing block gets cliffhanger's default (enabled, 30-day validity).
- Config migration 7->8 switches the old seed (`enabled` false, zero `enabled_at`) on once. A block
  the cloud disabled looks the same and is switched on too; the next cloud config poll disables it again.

## Impact
- Affected specs: `configuration`
- Affected code: `src/cmd/factory.go`, `src/internal/config/config.go`
