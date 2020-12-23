---
layout: default
---

# Config Dir

Generally pdfcpu does not have to be configured.

Yet it uses a configuration directory for user font management and storing the default configuration in effect.

pdfcpu will create this dir at the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) on execution of the first command.

You can look up its location like so:

```
Go-> pdfcpu ver -v
pdfcpu: v0.3.8 dev
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

<br>
This is also the location of config.yml.

This file holds carefully selected default values for various aspects of pdfcpu's operation:
```
Go-> cat config.yml
#########################
# Default configuration #
#########################

reader15: true
decodeAllStreams: false

# validationMode:
# ValidationStrict,
# ValidationRelaxed,
# ValidationNone
validationMode: ValidationRelaxed

# eol for writing:
# EolLF
# EolCR
# EolCRLF
eol: EolLF

writeObjectStream: true
writeXRefStream: true
encryptUsingAES: true

# encryptKeyLength: max 256
encryptKeyLength: 256

# permissions for encrypted files:
# -3901 = 0xF0C3 (PermissionsNone)
#    -1 = 0xFFFF (PermissionsAll)
permissions: -3901

# displayUnit:
# points
# inches
# cm
# mm
unit: cm
```

<br>

User fonts are installed using the [font install](../fonts/fonts_install.md) command.

pdfcpu also stores internal representations of installed user fonts in the config dir.

```
Go-> tree
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
