---
layout: default
title: "Resize"
---

# Resize

Resize selected pages of `inFile` by scale factor, to a standard paper size or to specific page dimensions.
Choose whether to preserve the requested page orientation and allow additional content rotation.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu resize description inFile [ outFile ] [flags]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](/getting_started/page_selection) | selected pages | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description          | required 
|:-------------|:---------------------|:---------
| description  | configuration string | yes
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

### Description

A configuration string with input parameters for the resize command.

| parameter           | values                                                        
|:--------------------|:------------------------------------------------------
| scalefactor         | 0.0 < s < 1.0 or s > 1.0           
| dimensions          | (width, height) in user units eg. '400 200'      
| enforce             | preserve requested output-page orientation: on/off true/false t/f
| formsize, papersize | [paper size](/paper) to be used. Append L or P to enforce landscape/portrait mode| f: A4
| rotate              | allow additional best-fit content rotation: on/off true/false t/f; default on
| border              | draw the fitted content boundary: on/off true/false t/f
| bgcolor             | [color](/getting_started/color)                  | none

<br>

`enforce:on` preserves the requested output-page orientation. The `P` and `L` paper-size suffixes also
select and enforce that orientation. Without enforcement, the destination follows the source orientation.

`rotate:on` is the default: when orientation is enforced, resize may turn content by 90 degrees for a better fit.
Use `rotate:off` to preserve its displayed orientation, scale proportionally and center it in the destination.
This does not suppress normalization of an existing page-level or inherited `/Rotate` value. Annotation rectangles
and quad points follow the same complete transform as the content. The option has no effect on scale-only resizing
or sizing by just width or height.

Direct Go callers can set `model.Resize.DisableContentRotation` to `true`. Its zero value retains best-fit rotation.

## Examples

Enlarge pages by doubling the page dimensions, keep orientation.
```sh
$ pdfcpu resize "scale:2" in.pdf out.pdf
```

<br>

Shrink first 3 pages by cutting in half the page dimensions, keep orientation.
```sh
$ pdfcpu resize "sc:.5" in.pdf out.pdf --pages 1-3
```

<br>

Resize pages to width of 40 cm, keep orientation.
```sh
$ pdfcpu resize -u cm -- "dim:40 0" in.pdf out.pdf
```

<br>

Resize pages to A4, keep orientation.
```sh
$ pdfcpu resize "form:A4" in.pdf out.pdf
```

<br>

Resize pages to A4 and enforce orientation (here: portrait mode), apply background color.
```sh
$ pdfcpu resize "f:A4P, bgcol:#d0d0d0" in.pdf out.pdf
```

<br>

Resize mixed portrait and landscape pages to A4 portrait while preserving displayed content orientation.
Landscape content fits the A4 width and is centered vertically, leaving unused space above and below.

```sh
$ pdfcpu resize 'form:A4P, rotate:off' in.pdf out.pdf
```

<br>

Resize pages to 400 x 200 points, keep orientation.
```sh
$ pdfcpu resize "dim:400 200" in.pdf out.pdf
```

<br>

Resize pages to 400 x 200 points, enforce orientation.
```sh
$ pdfcpu resize "dim:400 200, enforce:true" in.pdf out.pdf
```

<br>

Resize a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-design/spec-sheet.pdf - \
   | pdfcpu resize 'form:A4' - - \
   | aws s3 cp - s3://acme-design/spec-sheet-a4.pdf
```
