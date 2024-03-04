---
layout: default
---

# Reset Page Mode

This command resets the configured page mode for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode reset inFile
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

<br>

## Examples

Reset page mode for `test.pdf`:

```sh
$ pdfcpu pagemode reset test.pdf
$ pdfcpu pagemode list test.pdf
No page mode set, PDF viewers will default to "UseNone"
```