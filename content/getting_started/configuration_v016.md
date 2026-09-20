---
layout: default
title: "Configuration Reset Required in v0.16"
---

# Configuration Reset Required in v0.16

This guide covers Go API and file-backed configuration changes when upgrading from v0.15 or earlier to v0.16,
including v0.16.0-rc.1.
The module requires Go 1.26.0 or later. <br><br>
Once the candidate is published, select it explicitly with:<br>
`go get github.com/pdfcpu/pdfcpu@v0.16.0-rc.1`<br>
`@latest` may select an earlier stable release.

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

When a configuration file uses an older or newer schema than pdfcpu supports, loading returns an error that the application can identify and handle. Loading never modifies or resets the file automatically.

* Use `api.LoadConfiguration(api.ConfigurationOptions{})` and handle the returned configuration and error
* handle `ErrConfigurationResetRequired` and schema compatibility errors
* let the application or its operator decide when to call `ResetConfigurationWithOptions`.

Applications using stateless configuration need no configuration-file migration, but must still update changed API calls.
Passing `nil` loads the default configuration and may initialize files on disk. For an explicitly
stateless operation:

```go
conf, err := api.LoadConfiguration(api.ConfigurationOptions{
    Mode: api.ConfigurationModeStateless,
})
if err != nil {
    return err
}
return api.ValidateFile(ctx, "input.pdf", conf, nil)
```

This example assumes a non-nil `ctx` and imports `github.com/pdfcpu/pdfcpu/pkg/api`. Stateless mode provides
core PDF fonts and an empty local trust pool. Use a prepared read-only configuration when user fonts or trusted
certificates are required.

The no-argument `api.LoadConfiguration()` wrapper has been removed. Existing callers must pass configuration options
and handle the returned error.

A configuration passed to an operation remains owned by the application and can be reused after the call. Clone it before
applying different settings for another job; do not mutate it while concurrent operations use it.

## Operation contexts in v0.16

Long-running API operations now take a Go context as their first argument. Pass the context belonging to the request or
job so cancellation can stop the work safely. Use `context.Background()` when cancellation is not needed, and do not pass
a nil context.

For example, update a v0.15 caller:

```go
err := api.OptimizeFile("input.pdf", "output.pdf", conf)
```

To the v0.16 signature:

```go
err := api.OptimizeFile(ctx, "input.pdf", "output.pdf", conf, nil)
```

Use canonical operation names; interim context-free and `WithContext`/`WithOptions` variants were consolidated.
These are alternative call-site examples, not a single block to paste together.

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
