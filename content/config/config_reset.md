---
layout: default
title: "Reset Configuration"
---

# Reset Configuration

The configuration file (config.yml) holds carefully selected default values for various aspects of pdfcpu's operations.

This command replaces the selected configuration file with the built-in defaults and current configuration schema.

See also [initialize configuration](/config/config_init), [inspect configuration](/config/config_inspect) and the
[configuration directory](/getting_started/config_dir) documentation.


## Usage

```
pdfcpu config reset
pdfcpu config reset --force
```

Without `--force`, pdfcpu asks for confirmation. Use `--force` for non-interactive execution.

Warning: Back up `config.yml` before executing this command if you have custom settings.

## Background

Configuration schema versions are independent of pdfcpu release versions. A configuration-dependent command reports when
the selected schema is older than the one required by the running build and prints the applicable reset command. A newer
schema requires upgrading pdfcpu instead. Users upgrading an existing v0.15 or older configuration should follow
[Configuration Reset Required in v0.16](/getting_started/configuration_v016).

The command follows the normal root precedence and resets only the selected `config.yml`. It does not reset a stateless
configuration. It can replace an older, newer or malformed configuration because it does not load that file first. Fonts
and certificates stored beside it are preserved.

## Output

Interactive reset:

```
$ pdfcpu config reset
Reset the selected configuration to built-in defaults? (yes/no): yes
configuration reset
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
schema version: 1
```
