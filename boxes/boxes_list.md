---
layout: default
---

# List Boxes

* A page boundary is any of mediaBox, cropBox, trimBox, bleedBox, artBox.

* This command prints a list of selected page boundaries for selected pages using the display unit

Have a look at some [examples](#examples).

## Usage

```
pdfcpu boxes list [-p(ages) selectedPages] '[boxTypes]' inFile
```

<br>

### Flags

| flag                             | description     | required
|:---------------------------------|:----------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no


<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](getting_started/common_flags.md)          | user password   |
| [opw](getting_started/common_flags.md)          | owner password  |

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
pdfcpu box list test.pdf
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
pdfcpu box list -u cm -pages 1 'm,c' test.pdf
listing mediaBox, cropBox for test.pdf
pages: 1
Page 1:
  MediaBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (inherited)
   CropBox (cm) (0.00, 0.00, 21.00, 29.70) w=21.00 h=29.70 ar=0.71 (default)
```

<br>

## Hint

The [info](../info.md) command also prints page boundaries if the *-pages* flag is present.
