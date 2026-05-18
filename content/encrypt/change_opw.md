---
layout: default
title: "Change Owner Password"
---

# Change Owner Password

This command changes the password which is also known as the *set permissions password* or the *master password*. Have a look at some [examples](#examples).
 
## Usage

```
pdfcpu changeopw inFile opwOld opwNew [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| opwOld       | current owner password | yes
| opwNew       | new owner password     | yes, must not be empty!
| outFile      | PDF output file, use `-` to write to stdout | no

<br>

## Examples

You have to set the *owner password* when you `encrypt` a file and you can change it anytime later with `changeopw`.

Change the *owner password*:
```sh
$ pdfcpu encrypt enc.pdf --opw opw
writing enc.pdf ...

$ pdfcpu changeopw enc.pdf opw opwNew
writing enc.pdf ...
```

<br>

Change the owner password for a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-legal/client.pdf - \
   | pdfcpu changeopw --upw "$UPW" - "$OLD_OPW" "$NEW_OPW" - \
   | aws s3 cp - s3://acme-legal/client-rotated-opw.pdf
```
