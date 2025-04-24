---
layout: default
---

# Configuration

Generally pdfcpu does not have to be configured.

There is a configuration directory for user font management and storing the default configuration in effect.


## Config Dir

pdfcpu will create this directory at the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) on the very first execution of a pdfcpu command.

You can look up its location either like so:

```
$ pdfcpu version
pdfcpu: v0.10.2 dev
commit: c5014528 (2025-04-23T12:42:04Z)
base  : go1.24.2
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

`pdfcpu config list` will also print the config file path followed by its content.

Please check out the [config list](../config/config_list.md) command.

<br>

## Certificates

Certificates are needed for processing digital signatures.

pdfcpu preloads rootCAs for users in europe.
Additional certificates will also be located here.
You can import them using the [cert import](../core/certs.md) command.

Certificates are located in the dir tree below `certs`:

```
$ tree
..
├── certs
│   └── eu
│       ├── ades-lotl.p7c
│       ├── at.p7c
│       ├── be.p7c
│       ├── bg.p7c
│       ├── cy.p7c
│       ├── cz.p7c
│       ├── de.p7c
│       ├── dk.p7c
│       ├── ee.p7c
│       ├── el.p7c
│       ├── es.p7c
│       ├── euiba-tl.p7c
│       ├── eutl.p7c
│       ├── fi.p7c
│       ├── fr.p7c
│       ├── hr.p7c
│       ├── hu.p7c
│       ├── ie.p7c
│       ├── is.p7c
│       ├── it.p7c
│       ├── li.p7c
│       ├── lt.p7c
│       ├── lu.p7c
│       ├── lv.p7c
│       ├── mt.p7c
│       ├── nl.p7c
│       ├── no.p7c
│       ├── pl.p7c
│       ├── pt.p7c
│       ├── ro.p7c
│       ├── se.p7c
│       ├── si.p7c
│       ├── sk.p7c
│       ├── ua.p7c
│       └── uk.p7c
├── config.yml
└── fonts
```


## User Fonts

User fonts are installed using the [font install](../fonts/fonts_install.md) command.

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

Use the [-conf](common_flags.md) flag to set a custom config dir path.

You can also use this flag to disable the usage of a config dir.

This comes in handy in (serverless) environments where the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) is not defined - as long as you are not using user fonts.
