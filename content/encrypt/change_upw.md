---
layout: default
title: "Change User Password"
---

# Change User Password

This command changes the password which is also known as the *open doc password*. Have a look at some [examples](#examples).

## Usage

```
pdfcpu changeupw inFile upwOld upwNew [ outFile ] [flags]
pdfcpu changeupw inFile --upwold-file oldFile --upwnew-file newFile [ outFile ] [flags]
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

Both current passwords are authenticated before the change. Supply the current owner password with
`--opw` or `--opw-file`; omitting it tries an empty password. The old positional password (or its file alternative)
is the current user password being changed.

## Password files

Use both `--upwold-file` and `--upwnew-file` together, omitting the positional old and new passwords.
Use `--opw-file` to supply the other password from a file. Literal password strings remain supported.
See [Password files](/getting_started/common_flags/#password-files) for newline handling, empty passwords and conflicts.

```sh
pdfcpu changeupw input.pdf --opw-file /run/secrets/opw \
  --upwold-file /run/secrets/old-password --upwnew-file /run/secrets/new-password output.pdf
```

The argument table above describes the literal-password form.
An empty new-password file removes the password required to open the document.

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
   | pdfcpu changeupw - --opw "$OPW" "$OLD_UPW" "$NEW_UPW" - \
   | aws s3 cp - s3://acme-legal/client-rotated-upw.pdf
```
