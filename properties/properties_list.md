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

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all document properties of `in.pdf`:

```sh
$ pdfcpu properties list in.pdf
dept = hr
group = 3
```
