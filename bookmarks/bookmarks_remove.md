---
layout: default
---

# Remove Bookmarks

This command removes all bookmarks.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu bookmarks remove inFile [outFile]
```

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
| outFile      | PDF output file     | no

<br>

## Examples

 Remove all bookmarks:

```
$ pdfcpu bookmarks remove bookmarkSimple.pdf

$ pdfcpu bookmarks list bookmarkSimple.pdf
no bookmarks available
```
