---
layout: default
---

# Create Cheat Sheets

Some Unicode fonts cover thousands of Unicode code points.

For your reference pdfcpu is able to produce *cheat sheets*.

A cheat sheet is a single-page PDF file containing a grid mapping the Unicode code points of a Unicode plane to font glyphs.

The following command will produce one cheat sheet for each Unicode plane with code points covered in a font in the current dir:

```
pdfcpu font cheat Unifont-JPMedium
creating cheatsheets for: Unifont-JPMedium

ls Unifont-JP*
-rw-r--r--@ 1 horstrutter  staff   3.8M Nov  5 01:32 Unifont-JPMedium_BMP.pdf
-rw-r--r--  1 horstrutter  staff   3.7M Nov  5 01:32 Unifont-JPMedium_SIP.pdf
-rw-r--r--  1 horstrutter  staff   3.6M Nov  5 01:32 Unifont-JPMedium_SMP.pdf

```

The following command will produce cheat sheets for all user fonts installed:

```
pdfcpu font cheat
creating cheatsheets for: Roboto-Regular
creating cheatsheets for: STSong
creating cheatsheets for: STSongti-SC-Black
creating cheatsheets for: STSongti-SC-Bold
creating cheatsheets for: STSongti-SC-Light
creating cheatsheets for: STSongti-SC-Regular
creating cheatsheets for: STSongti-TC-Bold
creating cheatsheets for: STSongti-TC-Light
creating cheatsheets for: STSongti-TC-Regular
creating cheatsheets for: SimSun
creating cheatsheets for: Unifont-JPMedium
creating cheatsheets for: UnifontMedium
creating cheatsheets for: UnifontUpperMedium
```

