---
layout: default
---

# Remove Properties

This command removes properties from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties remove inFile [name...]
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| name...      | one or more property names | no

<br>

## Examples

Remove a specific property from `in.pdf`:

```sh
$ pdfcpu prop remove in.pdf dept
```

<br>

Remove all properties:

```sh
$ pdfcpu prop remove test.pdf
```