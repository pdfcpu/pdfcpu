---
layout: default
---

# Configuration

Generally pdfcpu does not have to be configured.

There is a configuration directory for user font management and storing the default configuration in effect.


## Config Dir

pdfcpu will create this directory at the default user's config directory on execution of the first command.

pdfcpu will create this directory at the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) on execution of the first command.

You can look up its location either like so:

```
$ pdfcpu version
pdfcpu: v0.9.0 dev
commit: 38b29927 (2024-10-16T21:08:47Z)
base  : go1.22.0
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

`pdfcpu config list` will also print the config file path followed by its content.

Please check out the [config list](../config/config_list.md) command.

<br>

## User Fonts

User fonts are installed using the [font install](../fonts/fonts_install.md) command.

pdfcpu also stores internal representations of installed user fonts in the config dir.

```
$ tree
.
├── config.yml
└── fonts
    ├── Roboto-Regular.gob
    ├── STSong.gob
    ├── STSongti-SC-Black.gob
    ├── STSongti-SC-Bold.gob
    ├── STSongti-SC-Light.gob
    ├── STSongti-SC-Regular.gob
    ├── STSongti-TC-Bold.gob
    ├── STSongti-TC-Light.gob
    ├── STSongti-TC-Regular.gob
    ├── SimSun.gob
    ├── Unifont-JPMedium.gob
    ├── UnifontMedium.gob
    └── UnifontUpperMedium.gob
```

Use the [-conf](common_flags.md) flag to set a custom config dir path.

You can also use this flag to disable the usage of a config dir.

This comes in handy in (serverless) environments where the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) is not defined - as long as you are not using user fonts.
