---
layout: default
title: "Configuration Reset Required in v0.16"
---

# Configuration Reset Required in v0.16

This guide applies when upgrading an existing file-backed pdfcpu configuration from v0.15 or earlier.

Configuration compatibility is determined by `schemaVersion` instead of the pdfcpu release version.
Future pdfcpu releases therefore require a reset only when the configuration schema changes.

New installations and stateless use with `--conf disable` require no migration.

When pdfcpu encounters a legacy configuration, it preserves the file and reports:

```text
configuration reset required
config: <path>/config.yml
detected schema version: legacy
required schema version: 1
run: pdfcpu config reset
```

Follow the workflow matching your environment.


## Local CLI with the default configuration

If you have not customized `config.yml`, reset it and validate the replacement:

```sh
pdfcpu config reset
pdfcpu config validate
```

`config reset` replaces `config.yml` after confirmation. Installed user fonts and trusted certificates are preserved.


## Customized configuration or explicit configuration root

Back up your existing `config.yml` before resetting it. Run the reset using the same configuration root reported by
pdfcpu:

```sh
pdfcpu --conf /srv/pdfcpu config reset
pdfcpu --conf /srv/pdfcpu config validate
pdfcpu --conf /srv/pdfcpu config inspect
```

Reapply any settings you still need to the newly generated file. Do not copy the legacy file back over it, because that
also restores the incompatible schema.


## Managed service or container

Prepare or reset the configuration during deployment while its volume is writable:

```sh
pdfcpu --conf /srv/pdfcpu config reset --force
pdfcpu --conf /srv/pdfcpu config validate
```

After validation, the configuration may be mounted read-only for normal operation. Do not rely on a running read-only
container to initialize or reset it. Ensure every pdfcpu invocation uses the same explicit configuration root.


## Stateless CLI

No migration is needed when pdfcpu runs without a configuration tree:

```sh
pdfcpu --conf disable validate in.pdf
```

Stateless mode ignores `config.yml` and cannot initialize or reset configuration resources.


## Go API application

Applications loading a file-backed configuration receive a typed compatibility error when the configuration requires
attention. Loading never modifies or resets the file automatically.

* Use `api.LoadConfiguration(api.ConfigurationOptions{})` and handle the returned configuration and error
* handle `ErrConfigurationResetRequired` and schema compatibility errors
* let the application or its operator decide when to call `ResetConfigurationWithOptions`.

Applications using stateless configuration require no migration.

The no-argument `api.LoadConfiguration()` wrapper has been removed. Existing callers must pass configuration options
and handle the returned error.

## Progress options in v0.16

`Validate`, `ValidateFile`, `ValidateFiles`, `Optimize`, `OptimizeFile` and `ReadValidateAndOptimize` accept a final
`*api.ProgressOptions` parameter. Pass `nil` when progress reporting is unnecessary:

```go
err := api.ValidateFile(ctx, "input.pdf", conf, nil)
```

To observe progress, pass a pointer:

```go
options := &api.ProgressOptions{
    Observer: func(event api.ProgressEvent) error {
        fmt.Println(event.Stage)
        return nil
    },
}
err := api.OptimizeFile(ctx, "input.pdf", "output.pdf", conf, options)
```

See [Configuration Modes](/config/config_modes) for access behavior,
[Configuration](/getting_started/config_dir) for root selection, or [config reset](/config/config_reset) for reset
behavior.
