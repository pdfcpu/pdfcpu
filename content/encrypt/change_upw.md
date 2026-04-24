---
layout: default
title: "Change User Password"
---

# Change User Password

This command changes the password which is also known as the *open doc password*. Have a look at some [examples](#examples).

## Usage

```
pdfcpu changeupw inFile upwOld upwNew [flags]
````

<br>

### [Common Flags](/getting_started/common_flags)

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
$ pdfcpu encrypt enc.pdf --upw upw --opw opw
writing enc.pdf ...

$ pdfcpu changeupw enc.pdf upw upwNew
writing enc.pdf ...
```

<br>

Set the *user password* of a document that has none. Whenever you change the *user password* of a document you also have to provide the current *owner password*:

```sh
$ pdfcpu encrypt enc.pdf --opw opw
writing enc.pdf ...

$ pdfcpu changeupw enc.pdf "" upwNew
Please provide the owner password with --opw

$ pdfcpu changeupw enc.pdf "" upwNew --opw opw
writing enc.pdf ...
```