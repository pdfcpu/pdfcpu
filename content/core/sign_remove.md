---
layout: default
title: "Remove Signatures"
---

# Remove Signatures

Remove digital signatures from a PDF.

```
pdfcpu signatures remove inFile [ outFile ] [flags]
```

## Flags

| name  | description | default | required |
|:------|:------------|:--------|:---------|
| rmenc | remove encryption while removing signatures | false | no |

## Arguments

| name    | description                                           | required |
|:--------|:------------------------------------------------------|:---------|
| inFile  | PDF input file, use `-` to read from stdin            | yes |
| outFile | PDF output file, use `-` to write to stdout           | no |

## Example

Remove signatures from a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures remove - - \
   | aws s3 cp - s3://acme-signing/executed-unsigned.pdf
```
