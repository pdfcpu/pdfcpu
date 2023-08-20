---
layout: default
---

# Add Properties

This command adds property name/value pairs to a PDF document. Have a look at some [examples](#examples).

You can also set the PDFs *Title*, *Subject* and *Author*. 

## Usage

```
pdfcpu properties add inFile nameValuePair...
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
| nameValuePair | 'name = value' | yes

<br>

## Examples

Adding a property:

```sh
$ pdfcpu properties add in.pdf name = value
```

```sh
$ pdfcpu properties add in.pdf 'name = value'
```

Adding two properties:
```sh
$ pdfcpu properties add in.pdf 'name1 = value1' 'name2 = value2'
```

Setting Title and Author:
```sh
$ pdfcpu properties add in.pdf 'Title = My title' 'Author = Me'
```
