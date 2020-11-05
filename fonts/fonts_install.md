---
layout: default
---

# Install Fonts

Install TrueType fonts for embedding text based stamps/watermarks.

In order to produce stamps/watermarks using your favorite TrueType font you need to install it as a user font:

```sh
pdfcpu font install SimSun.ttf
installing to /Users/horstrutter/Library/Application Support/pdfcpu/fonts...
SimSun
```

TrueType colections are also supported:

```sh
pdfcpu font install Songti.ttc
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


