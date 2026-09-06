---
layout: default
title: "Configuration Modes"
---

# Configuration Modes

A configuration mode controls how pdfcpu obtains settings and resources. It is separate from the configuration root,
which controls where file-backed configuration is stored.


## Automatic (`auto`)

Automatic mode is the normal CLI mode. It uses the selected file-backed configuration tree and may create missing
resources when a command needs them.

Go API callers select `ConfigurationModeAuto`; it is also the zero-value mode.

Use automatic mode for local CLI work or a service with a writable, managed configuration root.


## Stateless (`stateless`)

Stateless mode uses built-in settings and the 14 core PDF fonts without accessing configuration files, user fonts or
trusted certificates.

CLI users select it with `--conf disable`. Go API callers select `ConfigurationModeStateless`.

Use stateless mode for an immutable container or pipeline that does not need external configuration resources.


## Read-only (`read-only`)

Read-only mode loads an existing file-backed configuration tree without creating or changing anything. Go API callers
select `ConfigurationModeReadOnly` when the application owns provisioning.

There is no CLI flag that selects read-only mode for normal PDF commands. The CLI uses read-only loading internally for
`config inspect` and `config validate`, allowing these commands to diagnose configuration without modifying it.

In `config inspect` output, `mode: auto` describes what normal PDF commands would use for the same selection. The
inspection itself never writes.


## Mode and root are separate choices

Automatic and read-only modes require a configuration root. Selecting `--conf PATH` changes that location; it does not
change the mode. Stateless mode bypasses root selection entirely.

See [Configuration](/getting_started/config_dir) for root precedence, default locations and the configuration directory
layout. See [Configuration Workflows](/config/config_workflows) for common command sequences or
[Config](/config/config) for the command overview.
