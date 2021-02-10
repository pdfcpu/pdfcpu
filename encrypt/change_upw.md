---
layout: default
---

# Change User Password

This command changes the password which is also known as the *open doc password*. Have a look at some [examples](#examples).

## Usage

```
pdfcpu changeupw [-opw ownerpw] inFile upwOld upwNew
````

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)          | owner password   |

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file         | yes
| upwOld       | current user password  | yes
| upwNew       | new user password      | yes

<br>

## Examples

You can set the *user password* either when you `encrypt` a file or later with `changeupw`.

Change the *user password* of a document that already has one:
```sh
pdfcpu encrypt -upw upw enc.pdf
writing enc.pdf ...

pdfcpu changeupw enc.pdf upw upwNew
writing enc.pdf ...
```

<br>

Set the *user password* of a document that has none. Any encrypted PDF file has either one of the two passwords set. Whenever you change the *user password* of a document that has a *owner password* set, you have to provide the current *owner password*:

```sh
pdfcpu encrypt -opw opw enc.pdf
writing enc.pdf ...

pdfcpu changeupw enc.pdf "" upwNew
Please provide the owner password with -opw

pdfcpu changeupw -opw opw enc.pdf "" upwNew
writing enc.pdf ...
```