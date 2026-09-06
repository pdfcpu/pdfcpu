---
layout: default
title: "Configuration"
---

# Configuration

Generally pdfcpu does not have to be configured.

There is a configuration directory for certificate and user font management and storing the default configuration in effect.

See [Configuration Reset Required in v0.16](/getting_started/configuration_v016) when upgrading an existing v0.15 or
older installation.
The generated file records a configuration `schemaVersion`, not the pdfcpu version. A release upgrade therefore
requires a reset only when its required configuration schema changes.

See [Configuration Modes](/config/config_modes) for automatic, stateless and read-only access.
<br>See [Configuration Workflows](/config/config_workflows) for common command sequences.
<br>See [Environment Variables](/getting_started/environment_variables#configuration-location) for configuration-root
precedence and platform defaults.


## Config Dir

In automatic mode, a configuration-dependent command initializes this directory below the default
[user configuration directory](https://pkg.go.dev/os#UserConfigDir) when required.

Inspect the selected location without modifying it. The relevant path portion of the output is shown here; see
[config inspect](/config/config_inspect) for the complete text and JSON output contracts:

```
$ pdfcpu config inspect
mode: auto
source: os-default
...
  config:
    path: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
    available: true
    exists: true
    writable: true
...
```

`pdfcpu config list` will also print the config file path followed by its content.

Please check out the [config list](/config/config_list) command.

<br>

## Certificates

Certificates are needed for processing digital signatures.<br>
pdfcpu keeps trusted certificates in the configuration directory below `certs`.

* Standard builds start with an empty trusted certificate store.
* Builds created with `-tags pdfcpu_eutl` initialize this store with an embedded snapshot of EU Trusted List certificate bundles.

Use:

* [certificates list](/core/certs) to inspect the current store
* [certificates reset](/core/certs) to restore the store
* [certificates import](/core/certs) to add missing certificates

Certificates are located in the dir tree below `certs`:

```
$ tree
..
├── certs
│   ├── root-ca.pem
│   └── intermediate-ca.pem
├── config.yml
└── fonts
```


## User Fonts

User fonts are installed using the [fonts install](/fonts/fonts_install) command.

pdfcpu also stores internal representations of installed user fonts in the config dir.

```
$ tree
..
├── certs
├── config.yml
└── fonts
    ├── Roboto-Regular.gob
    ├── Unifont-JPMedium.gob
    ├── UnifontMedium.gob
    └── UnifontUpperMedium.gob
```

Use the [--conf](/getting_started/common_flags) flag to select a custom configuration root. The configuration tree is
stored in the `pdfcpu` directory below that root.

Use `--conf disable` to select stateless mode. This is useful when the default
[user configuration directory](https://pkg.go.dev/os#UserConfigDir) is unavailable or the process must not access
filesystem configuration. Stateless mode cannot use user fonts or the trusted certificate store.

Read-only is not selected with a CLI flag. The CLI uses read-only loading internally for `config inspect` and
`config validate`; Go API applications may select it explicitly for externally provisioned configuration.
