---
layout: default
---

# Install Fonts

Install TrueType fonts for embedding text based stamps/watermarks.

```sh
pdfcpu fonts install fontFiles...
```

where `fontFiles` is a list of TrueType font files with the extension `.ttf`.

## Font directory

Fonts are installed into the [user's config directory](https://golang.org/pkg/os/#UserConfigDir).
