---
layout: default
---

# Change Owner Password

This command changes the password which is also known as the *set permissions password* or the *master password*. Have a look at some [examples](#examples).
 
## Usage

```
usage: pdfcpu changeopw [-v(erbose)|vv] [-upw userpw] inFile opwOld opwNew
```

<br>

### Flags

| name                             | description     | required
|:---------------------------------|:----------------|:--------
| [verbose](../getting_started.md) | turn on logging | no
| [vv](../getting_started.md)      | verbose logging | no
| [upw](../getting_started.md)     | user password   | if set

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file         | yes
| opwOld       | current owner password | yes
| opwNew       | new owner password     | yes

<br>

## Examples

You can set the *owner password* either when you `encrypt` a file or later with `changeopw`.

Change the *owner password* of a document that already has one:
```sh
pdfcpu encrypt -opw opw enc.pdf
writing enc.pdf ...

pdfcpu changeopw enc.pdf opw opwNew
writing enc.pdf ...
```

<br>

Set the *owner password* of a document that has none. Any encrypted PDF file has either one of the two passwords set. Whenever you change the *owner password* of a document that has a *user password* set, you have to provide the current *user password*:

```sh
pdfcpu encrypt -upw upw enc.pdf
writing enc.pdf ...

pdfcpu changeopw enc.pdf "" opwNew
Please provide the user password with -upw

pdfcpu changeopw -upw upw enc.pdf "" opwNew
writing enc.pdf ...
```