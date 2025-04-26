---
layout: default
---

# Rotate

Rotate selected pages of `inFile` clockwise by a multiple of 90 degrees. Have a look at some [examples](#examples).

## Usage

```
pdfcpu rotate [-p(ages) selectedPages] inFile rotation [outFile]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description     | required | values
|:-------------|:----------------|:---------|:-
| inFile       | PDF input file  | yes      |
| rotation     | rotation angle  | yes      | -270, -180, -90, 90, 180, 270
| outFile      | PDF output file | no       |

<br>

## Examples

Rotate all pages of a PDF file clockwise by 90 degrees:

```sh
$ pdfcpu rotate test.pdf 90
```

<br>
Rotate the first two pages counter clockwise by 90 degrees:

```sh
$ pdfcpu rotate -pages 1-2 test.pdf -90
```