---
layout: default
---

# Fonts

Print the list of supported fonts.

```sh
pdfcpu fonts
```

## Supported Fonts

The Adobe Core Fontset consisting of the following 14 Type 1 fonts:

* Helvetica, Helvetica-Bold, Helvetica-Oblique, Helvetica-BoldOblique
* Times-Roman, Times-Bold, Times-Italic, Times-BoldItalic
* [Courier](https://github.com/pdfcpu/pdfcpu/blob/master/pkg/testdata/fontSamples/Courier.pdf), Courier-Bold, Courier-BoldOblique, Courier-Oblique
* Symbol
* ZapfDingbats

These fonts are supposed to be supported by PDF Readers and do not have to be embedded
by pdfcpu eg. during stamping or watermarking.
