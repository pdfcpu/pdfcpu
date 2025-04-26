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

### [Common Flags](../getting_started/common_flags)

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