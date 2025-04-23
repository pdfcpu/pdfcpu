---
layout: default
---

# Signatures



* Have a look at some [examples](#examples).


## Usage

```
pdfcpu signatures validate [-a(ll) -f(ull)] -- inFile
```

<br>

### Flags

| name   | description             | default | required
|:-------|:------------------------|-------------------
| a(ll)  | validate all signatures | false   |no
| f(ull) | comprehensive output    | false   |no

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

<br>

## Examples

coming up..