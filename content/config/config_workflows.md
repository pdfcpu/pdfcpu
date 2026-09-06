---
layout: default
title: "Configuration Workflows"
---

# Configuration Workflows

Choose the workflow matching your environment or task. See [Configuration Modes](/config/config_modes) if you first need
to decide between automatic, stateless and read-only access.


## Local CLI

If you do not care where configuration is stored, run pdfcpu normally:

```sh
pdfcpu validate document.pdf
```

The first configuration-dependent command initializes the default configuration tree if it is missing. `pdfcpu version`,
help and shell completion do not need configuration and therefore do not initialize it.

Use configuration commands only when needed:

1. Run `pdfcpu config list` to find and review the stored configuration.
2. Edit `config.yml` only when you need to change a default.
3. Run `pdfcpu config validate` after editing it.
4. Run `pdfcpu config inspect` when diagnosing paths or configured policy.

You do not need `config init` in this workflow. Automatic initialization already performs that job.


## Service, container or Kubernetes

Use an explicit configuration root when configuration must live outside the operating system's user configuration
directory. See [Configuration](/getting_started/config_dir) for root selection and directory layout.

For a new writable root, initialize it once:

```sh
pdfcpu --conf /srv/app-config config init
```

Apply the required changes to `/srv/app-config/pdfcpu/config.yml`, then verify and use the selected configuration:

```sh
pdfcpu --conf /srv/app-config config validate
pdfcpu --conf /srv/app-config config inspect
pdfcpu --conf /srv/app-config validate document.pdf
```

Every invocation must select the same root. <br>
You can set `PDFCPU_CONFIG_ROOT=/srv/app-config` instead of repeating
`--conf`; an explicit `--conf` value takes precedence.

For a read-only mount, prepare and validate the complete tree before deployment. Do not run `config init` or
`config reset` inside the read-only container. Mount the parent root, not `config.yml` alone, when user fonts or trusted
certificates are part of the deployment.


## Non-modifying diagnostics

Validate an existing configuration and inspect the selected paths and configured policy without changing anything:

```sh
pdfcpu --conf /srv/app-config config validate
pdfcpu --conf /srv/app-config config inspect
pdfcpu --conf /srv/app-config config inspect --json
```

These commands use read-only loading internally and do not initialize a missing tree. JSON inspection is useful for
readiness checks and other automation.


## Immutable or configuration-free execution

Use stateless mode when a container or pipeline needs only built-in settings and the 14 core PDF fonts:

```sh
pdfcpu --conf disable config inspect
pdfcpu --conf disable validate document.pdf
```

Stateless mode does not discover, create or read a configuration tree. User fonts and the trusted certificate store are
unavailable. `config init` and `config reset` therefore do not apply.


## After editing config.yml

Validate the file, then inspect the resulting runtime policy:

```sh
pdfcpu config validate
pdfcpu config inspect
```

When using a managed root, add the same `--conf PATH` used by the application.


## After a configuration schema change

When the stored schema is too old, a configuration-dependent command reports `configuration reset required` and prints
the exact reset command for the selected root.

Back up custom values, run the printed reset command, reapply the settings that are still relevant, then run
`config validate` and `config inspect`.
<br><br>
`config init` is not an upgrade command and does not replace an incompatible
`config.yml`.

Users upgrading an existing v0.15 or older configuration should follow
[Configuration Reset Required in v0.16](/getting_started/configuration_v016).
