---
layout: default
---

# Remove Attachments

This command removes properties from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties remove [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [name...]
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | verbose logging   | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

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
pdfcpu prop remove in.pdf dept
```

<br>

Remove all properties:

```sh
pdfcpu prop remove test.pdf
```