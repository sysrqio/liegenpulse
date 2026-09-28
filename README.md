# liegenpulse

EnEfG readiness audit CLI for municipal and public-sector building meter configurations. Validates `meters.yaml`, scores coverage and reachability (simulate mode uses an embedded fixture—no live Modbus or network egress), generates Telegraf input snippets, and exports Kom.EMS-style CSV for downstream EMS tools.

## Requirements

- Go 1.22+

## Install

```bash
go install github.com/sysrqio/liegenpulse/cmd/liegenpulse@latest
```

Or build from source:

```bash
make build
./bin/liegenpulse version
```

## Quick start

```bash
liegenpulse audit --config meters.yaml --simulate --threshold 70
liegenpulse audit --config meters.yaml --output json
liegenpulse generate --config meters.yaml
liegenpulse export --config meters.yaml --output file:export.csv
```

## Configuration (`meters.yaml`)

```yaml
site: My Municipality
buildings:
  - id: building-1
    name: Town hall
    has_heat: true
    meters:
      - id: electric-1
        type: electricity
        protocol: modbus
        address: "192.168.1.10:502"
        carrier: "Grid operator"
      - id: heat-1
        type: heat
        protocol: mqtt
        topic: org/meters/building-1/heat
        carrier: "District heat"
```

### Meter types

`electricity`, `heat`, `gas`, `water`

### Protocols

- **modbus** — requires `address` (e.g. `host:502`)
- **mqtt** — requires `topic`

Optional `carrier` improves the readiness score (metadata completeness).

## Commands

### `audit`

Scores EnEfG-oriented readiness (0–100) from meter coverage, simulated reachability, and carrier metadata.

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `meters.yaml` | Config path |
| `--output` | `stdout` | `stdout`, `json`, or `file:path` |
| `--simulate` | `true` | Use embedded reachability fixture (no network) |
| `--threshold` | `70` | Pass threshold |
| `--verbose` | `false` | Include per-meter details |

**Exit codes:** `0` score ≥ threshold, `1` invalid config, `2` score &lt; threshold

### `generate`

Prints Telegraf `inputs.modbus` and `inputs.mqtt_consumer` snippets derived from the config.

```bash
liegenpulse generate --config meters.yaml --output file:telegraf.conf
```

### `export`

Semicolon-separated CSV compatible with common Kom.EMS import workflows. Uses `--audit-json` or runs a simulate audit by default.

```bash
liegenpulse audit --config meters.yaml --output file:audit.json
liegenpulse export --config meters.yaml --audit-json audit.json --output file:meters.csv
```

### `version`

Prints the CLI version.

## Development

```bash
make test
make build
```

## License

MIT — see [LICENSE](LICENSE).
