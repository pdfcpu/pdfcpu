---
layout: default
---

# Multifill form via JSON or CSV

This command fills form fields with data via JSON or CSV.

Have a look at some [examples](#examples). 

## Usage

```
pdfcpu form multifill [-m(ode) single|merge] inFile inFileData outDir [outName]
```
<br>

### Flags

| name                             | description               | required   | values
|:---------------------------------|:--------------------------|:-----------|:-
| m(ode)                           | output mode (defaults to single) | no         | single, merge

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

| name         | description                        | required
|:-------------|:-----------------------------------|:--------
| inFile       | PDF input file containing form     | yes
| inFileData   | JSON/CSV input file with form data | yes
| outDir       | output directory                   | yes
| outName      | output file name                   | yes

<br>

## Examples
