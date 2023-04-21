---
layout: default
---

# List Images

* This command prints a list of embedded images for selected pages

Have a look at some [examples](#examples).

## Usage

```
pdfcpu images list [-p(ages) selectedPages] inFile
```

### Flags

| name                             | description     | required
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
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all embedded images of test.pdf:

 ```
pdfcpu image list test.pdf
pages: all
4 images available
Page  Obj# | Id  | Type  SoftMask ImgMask | Width | Height | Colorspace Comp bpc Interp |   Size | Filters
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    1     3 | Im0 | image    *             |  1667 |   2646 |  DeviceRGB    3   8        | 787 KB | DCTDecode
    2    10 | Im0 | image    *             |  1667 |   2646 | DeviceGray    1   8        | 1.6 MB | FlateDecode
    3     8 | Im0 | image    *             |  1667 |   2646 |  DeviceRGB    3   8        | 1.7 MB | FlateDecode
    4     9 | Im0 | image    *             |  1667 |   2646 |  DeviceRGB    3   8    *   | 3.8 MB | FlateDecode
```