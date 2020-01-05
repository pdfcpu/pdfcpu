---
layout: default
---

# Add Properties

This command adds property name/value pairs to a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties add [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile nameValuePair...
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | quiet mode        | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| nameValuePair | 'name = value' | yes

<br>

## Examples

Adding a property:

```sh
pdfcpu properties add in.pdf name = value
```

```sh
pdfcpu properties add in.pdf 'name = value'
```

Adding two properties:
```sh
pdfcpu properties add in.pdf 'name1 = value1' 'name2 = value2'
```
