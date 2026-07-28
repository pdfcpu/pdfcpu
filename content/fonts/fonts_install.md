---
layout: default
title: "Install Fonts"
---

# Install Fonts

Install TrueType fonts for embedding text based stamps/watermarks.

## Usage

```
pdfcpu fonts install fontFiles... [flags]
```

Supported inputs are TrueType fonts (`.ttf`) and TrueType collections (`.ttc`).
Installing a collection is transactional: pdfcpu validates and stages every member before replacing installed font data.
If any member fails, no member from that collection remains installed.

In order to produce stamps/watermarks using your favorite TrueType font you need to install it as a user font:

```sh
$ pdfcpu fonts install SimSun.ttf
installing to /Users/horstrutter/Library/Application Support/pdfcpu/fonts...
SimSun
```

TrueType collections are also supported:

```sh
$ pdfcpu fonts install Songti.ttc
installing to /Users/horstrutter/Library/Application Support/pdfcpu/fonts...
STSongti-SC-Black
STSongti-SC-Bold
STSongti-TC-Bold
STSongti-SC-Light
STSong
STSongti-TC-Light
STSongti-SC-Regular
```

## Font directory

Fonts are installed into the [user's config directory](https://golang.org/pkg/os/#UserConfigDir).
