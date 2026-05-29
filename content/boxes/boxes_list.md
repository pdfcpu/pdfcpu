---
layout: default
title: "List Boxes"
---

# List Boxes

* A page boundary is any of mediaBox, cropBox, trimBox, bleedBox, artBox.

* This command prints a list of selected page boundaries for selected pages using the display unit

Have a look at some [examples](#examples).

## Usage

```
pdfcpu boxes list [ boxTypes ] inFile [flags]
```

<br>

### Flags

| name                             | description     | required
|:---------------------------------|:----------------|---------
| [p(ages)](/getting_started/page_selection) | selected pages | no


<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| boxTypes     | comma separated list of box types: m(edia), c(rop), t(rim), b(leed), a(rt)      | no
| inFile       | PDF input file, use `-` to read from stdin      | yes

<br>

## Examples

 List all page boundaries of test.pdf:

```
$ pdfcpu boxes list test.pdf
listing mediaBox, cropBox, trimBox, bleedBox, artBox for upc.pdf
pages: all
Page 1: rot=+0 orientation:portrait
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (inherited)
   CropBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
   TrimBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
  BleedBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
    ArtBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)

Page 2: rot=+0 orientation:portrait
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (inherited)
   CropBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
   TrimBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
  BleedBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
    ArtBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
```

<br>

List mediaBox and cropBox of page 1 of test.pdf using display unit cm:

```
$ pdfcpu boxes list 'm,c' test.pdf --pages 1 --unit cm
listing mediaBox, cropBox for test.pdf
pages: 1
Page 1: rot=+0 orientation:portrait
  MediaBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (inherited)
   CropBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (default)
```

<br>

## Hint

The [info](/info) command also prints page boundaries if the *--pages* flag is present.

<br>

List page boundaries for a streamed PDF:

```sh
$ aws s3 cp s3://acme-print/ad.pdf - \
   | pdfcpu boxes list -
```
