---
layout: default
title: "Inspect Configuration"
---

# Inspect Configuration

Report the selected configuration paths, schema compatibility, network policy and resource limits without modifying the
configuration tree.


## Usage

```
pdfcpu config inspect
pdfcpu config inspect --json
```

Use `--conf PATH` to inspect a specific configuration root or `--conf disable` to inspect stateless mode.

For a file-backed selection, inspection resolves the normal configuration root and loads the existing tree read-only. It
does not initialize missing resources. An older schema produces reset guidance; a newer schema requires a newer pdfcpu
version. In either case, inspection writes nothing and produces no partial result.


## Summary fields

The first five fields describe how configuration was selected:

| Field | Meaning |
|:------|:--------|
| `mode` | How normal operations use the selected configuration: `auto` for file-backed configuration or `stateless` for `--conf disable`. Inspection itself does not initialize or write configuration. |
| `source` | How the root was selected: `flag`, `environment` or `os-default`. Stateless selection also reports `flag`. |
| `default` | `true` when `source` is `os-default`. This convenience field avoids interpreting `source`. |
| `stateless` | `true` when pdfcpu uses built-in configuration without filesystem resources. |
| `write capable` | `true` when normal operations in the reported mode may write configuration. Inspection itself never writes, and this field does not assert that the reported paths are writable. |

The `schema version` section reports the detected and supported `config.yml` schema versions. The `writable` field below
each path reports actual access for the current process. The network and limit sections contain configured values; they do
not include per-operation command-line overrides such as `--offline`.

The CLI does not expose a flag that selects `read-only` for normal PDF commands. See
[Configuration Modes](/config/config_modes) for how CLI and Go API users select each mode.


## Text output

Byte and pixel limits use compact units when the value is an exact multiple, for example `512 MB` and `100 MP`.

Example for a compatible configuration using the macOS default root:

```
$ pdfcpu config inspect
mode: auto
source: os-default
default: true
stateless: false
write capable: true
  root:
    path: /Users/horstrutter/Library/Application Support
    available: true
    exists: true
    writable: true
  config:
    path: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
    available: true
    exists: true
    writable: true
  fonts:
    path: /Users/horstrutter/Library/Application Support/pdfcpu/fonts
    available: true
    exists: true
    writable: true
  certificates:
    path: /Users/horstrutter/Library/Application Support/pdfcpu/certs
    available: true
    exists: true
    writable: true
schema version:
  detected: 1
  minimum supported: 1
  maximum supported: 1
network:
  offline: false
  HTTP timeout seconds: 5
  CRL timeout seconds: 10
  OCSP timeout seconds: 10
  preferred revocation checker: CRL
  allowed revocation hosts: (none)
limits:
  max input bytes: unlimited
  max object bytes: 64 MB
  max stream bytes: 512 MB
  max decode bytes: 512 MB
  max image pixels: 100 MP
  max image bytes: 512 MB
  max object count: 10000000
  max object stream count: 1000000
  max object stream first: 16 MB
  max xref entries: 10000000
  max recursion depth: 100
```

Path availability and writability reflect the current process and filesystem.


## JSON output

`--json` emits the same inspection result as tab-indented JSON. Resource limits remain numeric for machine processing.

```
{
	"mode": "auto",
	"source": "os-default",
	"default": true,
	"stateless": false,
	"writeCapable": true,
	"root": {
		"path": "/Users/horstrutter/Library/Application Support",
		"available": true,
		"exists": true,
		"writable": true
	},
	"paths": {
		"config": {
			"path": "/Users/horstrutter/Library/Application Support/pdfcpu/config.yml",
			"available": true,
			"exists": true,
			"writable": true
		},
		"fonts": {
			"path": "/Users/horstrutter/Library/Application Support/pdfcpu/fonts",
			"available": true,
			"exists": true,
			"writable": true
		},
		"certificates": {
			"path": "/Users/horstrutter/Library/Application Support/pdfcpu/certs",
			"available": true,
			"exists": true,
			"writable": true
		}
	},
	"schema": {
		"detected": 1,
		"minimumSupported": 1,
		"maximumSupported": 1
	},
	"network": {
		"offline": false,
		"httpTimeoutSeconds": 5,
		"crlTimeoutSeconds": 10,
		"ocspTimeoutSeconds": 10,
		"preferredRevocationChecker": "CRL",
		"allowedRevocationHosts": []
	},
	"limits": {
		"maxInputBytes": 0,
		"maxObjectBytes": 67108864,
		"maxStreamBytes": 536870912,
		"maxDecodeBytes": 536870912,
		"maxImagePixels": 100000000,
		"maxImageBytes": 536870912,
		"maxObjectCount": 10000000,
		"maxObjectStreamCount": 1000000,
		"maxObjectStreamFirst": 16777216,
		"maxXRefEntries": 10000000,
		"maxRecursionDepth": 100
	}
}
```

## Limits and network policy

See [Limits](/config/config_limits) for defaults, units, scope and Go API settings, in `config.yml` order.
The page also explains structural limits reported by inspection that are not YAML configuration keys.
