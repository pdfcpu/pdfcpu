---
layout: default
---

# List Boxes

A page boundary is any of mediaBox, cropBox, trimBox, bleedBox, artBox.

This command prints a list of selected page boundaries for selected pages using the display unit

Have a look at some [examples](#examples).

## Usage

```
usage: pdfcpu boxes list [-pages selectedPages] [-upw userpw] [-opw ownerpw] ['boxTypes'] inFile
```

<br>

### Flags

| flag                             | description     | required
|:---------------------------------|:----------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no
| [opw](../getting_started/common_flags.md)      | owner password | no
| [upw](../getting_started/common_flags.md)   | user password      | no

<br>

### Common Flags

| flag                             | description     | required
|:---------------------------------|:----------------|---------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)      | verbose logging | no
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      | no
| [u(nit)](../getting_started/common_flags.md) | display unit  | no
<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| boxTypes     | comma separated list of box types: m(edia), c(rop), t(rim), b(leed), a(rt)      | no
| inFile       | PDF input file      | yes

<br>

## Examples

 List all page boundaries of test.pdf:

```
Go-> pdfcpu box list test.pdf
listing mediaBox, cropBox, trimBox, bleedBox, artBox for upc.pdf
pages: all
Page 1:
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (inherited)
   CropBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
   TrimBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
  BleedBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
    ArtBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)

Page 2:
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (inherited)
   CropBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
   TrimBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
  BleedBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
    ArtBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71 (default)
```

<br>

List mediaBox and cropBox of page 1 of test.pdf using display unit cm:

```
Go-> pdfcpu box list -u cm -pages 1 'm,c' test.pdf
listing mediaBox, cropBox for test.pdf
pages: 1
Page 1:
  MediaBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (inherited)
   CropBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (default)
```