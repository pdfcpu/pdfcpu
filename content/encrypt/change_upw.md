---
layout: default
title: "Change User Password"
---

# Change User Password

This command changes the password which is also known as the *open doc password*. Have a look at some [examples](#examples).

## Usage

```
pdfcpu changeupw inFile upwOld upwNew [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| upwOld       | current user password  | yes
| upwNew       | new user password      | yes
| outFile      | PDF output file, use `-` to write to stdout | no

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

<br>

Change the user password for a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-legal/client.pdf - \
   | pdfcpu changeupw --opw "$OPW" - "$OLD_UPW" "$NEW_UPW" - \
   | aws s3 cp - s3://acme-legal/client-rotated-upw.pdf
```
