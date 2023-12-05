---
layout: default
---

# List Viewer Preferences

This command outputs a list of any configured viewer preferences.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu viewerpref list [-a(ll)] [-j(son)] inFile
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

| name    | description         | required
|:--------|:--------------------|:--------------------------
| all     | output all (including default values)      | no
| json    | output JSON                                | no
| inFile  | PDF input file                             | yes



<br>

## Examples

```sh
$ pdfcpu viewerpref list test.pdf
```

<br>

```sh
$ pdfcpu viewerpref list -all test.pdf
```

<br>

```sh
$ pdfcpu viewerpref list -json test.pdf
```

<br>

```sh
$ pdfcpu viewerpref list -all -json test.pdf
```

