---
layout: default
---

# List Fonts

Print the total list of supported fonts and user fonts.

```sh
pdfcpu fonts list
```

## Supported Fonts

The Adobe Core Fontset consisting of the following 14 Type 1 fonts:

* [Helvetica](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Helvetica.pdf)
, [Helvetica-Bold](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Helvetica-Bold.pdf)
, [Helvetica-Oblique](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Helvetica-Oblique.pdf)
, [Helvetica-BoldOblique](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Helvetica-BoldOblique.pdf)
* [Times-Roman](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Times-Roman.pdf)
, [Times-Bold](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Times-Bold.pdf)
, [Times-Italic](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Times-Italic.pdf)
, [Times-BoldItalic](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Times-BoldItalic.pdf)
* [Courier](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Courier.pdf)
, [Courier-Bold](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Courier-Bold.pdf)
, [Courier-BoldOblique](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Courier-BoldOblique.pdf)
, [Courier-Oblique](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Courier-Oblique.pdf)
* [Symbol](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/Symbol.pdf)
* [ZapfDingbats](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/samples/fonts/core/ZapfDingbats.pdf)

These fonts are supposed to be supported by PDF Readers and do not have to be embedded
by pdfcpu eg. during stamping or watermarking.

## User Fonts

Any TrueType font installed via `pdfcpu fonts install`.

```sh
pdfcpu font list
Corefonts:
  Courier
  Courier-Bold
  Courier-BoldOblique
  Courier-Oblique
  Helvetica
  Helvetica-Bold
  Helvetica-BoldOblique
  Helvetica-Oblique
  Symbol
  Times-Bold
  Times-BoldItalic
  Times-Italic
  Times-Roman
  ZapfDingbats

Userfonts(/Users/horstrutter/Library/Application Support/pdfcpu/fonts):
  STSong (43033 glyphs)
  STSongti-SC-Black (8535 glyphs)
  STSongti-SC-Bold (43033 glyphs)
  STSongti-SC-Light (43033 glyphs)
  STSongti-SC-Regular (43033 glyphs)
  STSongti-TC-Bold (43033 glyphs)
  STSongti-TC-Light (43033 glyphs)
  STSongti-TC-Regular (43033 glyphs)
  SimSun (22141 glyphs)
  ```
