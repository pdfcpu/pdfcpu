---
layout: default
---

# Add Keywords

This command adds keywords or key phrases to a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords add inFile keyword...
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
| keyword      | search keyword or keyphrase | yes

<br>

## Examples

Adding a key phrase and a keyword.
Put key phrases under single quotes:

```sh
$ pdfcpu keywords add in.pdf 'Tom Sawyer' classic
```
