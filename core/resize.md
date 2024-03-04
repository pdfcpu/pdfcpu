---
layout: default
---

# Resize

Resize selected pages of `inFile` either by scale factor, to standard form or specific page dimensions and optional enforce orientation.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu resize [-p(ages) selectedPages] -- [description] inFile [outFile]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description          | required 
|:-------------|:---------------------|:---------
| description  | configuration string | yes
| inFile       | PDF input file       | yes
| outFile      | PDF output file      | no

<br>

### Description

A configuration string with input parameters for the resize command.

| parameter           | values                                                        
|:--------------------|:------------------------------------------------------
| scalefactor         | 0.0 < s < 1.0 or s > 1.0           
| dimensions          | (width, height) in user units eg. '400 200'      
| enforce             | new aspect ratio: on/off true/false               
| formsize, papersize | [paper size](../paper.md) to be used. Append L or P to enforce landscape/portrait mode| f: A4
| bgcolor             | [color](../getting_started/color.md)                  | none

<br>

## Examples

Enlarge pages by doubling the page dimensions, keep orientation.
```sh
$ pdfcpu resize "scale:2" in.pdf out.pdf
```

<br>

Shrink first 3 pages by cutting in half the page dimensions, keep orientation.
```sh
$ pdfcpu resize -pages 1-3 -- "sc:.5" in.pdf out.pdf
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

Resize pages to A4 and enforce orientation(here: portrait mode), apply background color.
```sh
$ pdfcpu resize "f:A4P, bgcol:#d0d0d0" in.pdf out.pdf
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