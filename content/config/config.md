---
layout: default
title: "Config"
---

# Config

pdfcpu can manage its runtime configuration from the command line.

Most local CLI users do not need to set anything up. The first configuration-dependent command creates the default
configuration when necessary.


## Configuration at a glance

- [Configuration Modes](/config/config_modes) explains automatic, stateless and read-only access.
- [Configuration](/getting_started/config_dir) explains configuration roots, default locations and directory contents.
- [Configuration Workflows](/config/config_workflows) provides command sequences for common environments and tasks.


## Command overview

| Command | Use it to | Writes configuration? |
|:--------|:----------|:----------------------|
| [`pdfcpu config init`](/config/config_init) | Create missing configuration resources without replacing existing files. | Only creates missing resources. |
| [`pdfcpu config list`](/config/config_list) | Print the selected `config.yml` exactly as stored. | May initialize a missing tree in automatic mode. |
| [`pdfcpu config inspect`](/config/config_inspect) | Diagnose selected paths, schema, network policy and resource limits. | No |
| [`pdfcpu config inspect --json`](/config/config_inspect#json-output) | Produce the same inspection result as formatted JSON for automation. | No |
| [`pdfcpu config validate`](/config/config_validate) | Check the complete selected configuration after editing or provisioning it. | No |
| [`pdfcpu config reset`](/config/config_reset) | Replace `config.yml` with built-in defaults while preserving fonts and certificates. Use `--force` to skip confirmation. | Yes |

`config list` shows what is stored in `config.yml`. `config inspect` shows which configuration pdfcpu selected and which
runtime policy it defines.

Users upgrading an existing v0.15 or older configuration should follow
[Configuration Reset Required in v0.16](/getting_started/configuration_v016).

See [Limits](/config/config_limits) for network timeouts, input and resource limits in `config.yml` order.
