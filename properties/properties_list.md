---
layout: default
---

# List Properties

This command outputs a list of all properties. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties list inFile
```

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
| inFile       | PDF input file      | yes

<br>

## Examples

 List all document properties of `in.pdf`:

```sh
pdfcpu properties list in.pdf
dept = hr
group = 3
```
