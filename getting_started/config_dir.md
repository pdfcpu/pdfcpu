---
layout: default
---

# Config Dir

Generally pdfcpu does not have to be configured.

Yet it uses a configuration directory for user font management and storing the default configuration in effect.

pdfcpu will create this dir at the default [user's config directory](https://golang.org/pkg/os/#UserConfigDir) on execution of the first command.

You can look up its location either like so:

```
$ pdfcpu version
pdfcpu: v0.9.0 dev
commit: 38b29927 (2024-10-16T21:08:47Z)
base  : go1.22.0
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

or you can do the following which will print out both the config file path and its content. This file holds carefully selected default values for various aspects of pdfcpu's operations:

```
$ pdfcpu config
#############################
# pdfcpu v0.9.0 dev         #
# Created: 2024-10-16 13:49 #
#############################
#   Default configuration   #
#############################

# toggle for inFilename extension check (.pdf)
checkFileNameExt: true

reader15: true

decodeAllStreams: false

# validationMode:
# ValidationStrict,
# ValidationRelaxed,
validationMode: ValidationRelaxed

# validate cross reference table right before writing.
postProcessValidate: true

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
# 0xF0C3 (PermissionsNone)
# 0xF8C7 (PermissionsPrint)
# 0xFFFF (PermissionsAll)
# See more at model.PermissionFlags and PDF spec table 22
permissions: 0xF0C3

# displayUnit:
# points
# inches
# cm
# mm
unit: points

# timestamp format: yyyy-mm-dd hh:mm
# Switch month and year by using: 2006-02-01 15:04
# See more at https://pkg.go.dev/time@go1.17.1#pkg-constants
timestampFormat: 2006-01-02 15:04

# date format: yyyy-mm-dd
dateFormat: 2006-01-02

# toggle optimization
optimize: true

# optimize page resources via content stream analysis.
optimizeResourceDicts: true

# optimize duplicate content streams across pages.
optimizeDuplicateContentStreams: false

# merge creates bookmarks.
createBookmarks: true

# viewer is expected to supply appearance streams for form fields.
needAppearances: false

# internet availability.
offline: false

# http timeout in seconds.
timeout: 5
```

<br>

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
