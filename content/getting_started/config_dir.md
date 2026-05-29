---
layout: default
title: "Configuration"
---

# Configuration

Generally pdfcpu does not have to be configured.

There is a configuration directory for certificate and user font management and storing the default configuration in effect.


## Config Dir

pdfcpu will create this directory at the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) on the very first execution of a pdfcpu command.

You can look up its location either like so:

```
$ pdfcpu version
pdfcpu: v0.12.0 dev
commit: adbc7ca2 (2026-04-03T17:15:58Z)
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
base  : go1.26.1
```

`pdfcpu config list` will also print the config file path followed by its content.

Please check out the [config list](/config/config_list) command.

<br>

## Certificates

Certificates are needed for processing digital signatures.

The pdfcpu binary does not bundle trusted certificates.
pdfcpu keeps trusted certificates in the configuration directory below `certs`.
You can inspect the current store with [certificates list](/core/certs), restore the store with [certificates reset](/core/certs), and add missing certificates with [certificates import](/core/certs).

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

User fonts are installed using the [font install](/fonts/fonts_install) command.

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

Use the [--conf](/getting_started/common_flags) flag to set a custom config dir path.

You can also use this flag to disable the usage of a config dir.

This comes in handy in (serverless) environments where the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) is not defined - as long as you are not using user fonts or the certificate store.
