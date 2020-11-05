---
layout: default
---

# Fonts

Manage user fonts.

In addition to the standard PDF core font set you are also free to use
your favorite TrueType font for stamping/watermarking your PDF files.

In order to do this first you need to install your font as a pdfcpu user font.
pdfcpu keeps an internal representation of the font in the pdfcpu config dir.

Supported are simple TrueType font files (*.ttf) and TrueType collections (*.ttc).
OpenType font files (*.otf) are not supported at the moment.

pdfcpu supports the whole Unicode code range covering the 65536 Unicode code points of the Basic Multilingual Plane (BMP)
plus all defined code points of the defined supplementary planes (1, 2, 14-16) and any code points yet to be defined.
This means pdfcpu also allows you to stamp PDFs using your favorite CJKV based font on the command line or via the api.

There are also commands for listing all installed fonts and producing cheat sheets.







