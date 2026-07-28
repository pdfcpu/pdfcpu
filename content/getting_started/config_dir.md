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
version: v0.14.0-rc.1 dev
 config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
 commit: f08edb2c
   date: 2026-07-27 20:44:48 UTC
     go: go1.26.5
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

Use the [--conf](/getting_started/common_flags) flag to set a custom config dir path.

You can also use this flag to disable the usage of a config dir.

This comes in handy in (serverless) environments where the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) is not defined - as long as you are not using user fonts or the certificate store.
