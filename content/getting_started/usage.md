---
layout: default
title: "Usage"
---

# Usage

Use:

    pdfcpu [command] --help

for detailed information about a specific command.

Commands are grouped by domain:

## [Document](/core/validate)

Inspect, create, validate, optimize, combine, and reshape complete PDF files.

```
pdfcpu validate inFile... [flags]
pdfcpu optimize inFile [ outFile ] [flags]
pdfcpu info     inFile... [flags]
pdfcpu create   inFileJSON [ inFile ] outFile [flags]
pdfcpu merge    outFile inFile... [flags]
pdfcpu split    inFile outDir [ span | pageNr... ] [flags]
pdfcpu trim     inFile [ outFile ] [flags]
pdfcpu collect  inFile [ outFile ] [flags]
```

## [Page](/pages/page)

Work with pages and page geometry.

```
pdfcpu pages insert [ description ] inFile [ outFile ] [flags]
pdfcpu pages remove inFile [ outFile ] [flags]

pdfcpu rotate       inFile rotation [ outFile ] [flags]
pdfcpu nup          [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu booklet      [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu resize       description inFile [ outFile ] [flags]
pdfcpu poster       description inFile outDir [ outFile ] [flags]
pdfcpu ndown        [ description ] n inFile outDir [ outFile ] [flags]
pdfcpu cut          description inFile outDir [ outFile ] [flags]
pdfcpu crop         description inFile [ outFile ] [flags]
pdfcpu zoom         description inFile [ outFile ] [flags]

pdfcpu boxes list   [ boxTypes ] inFile [flags]
pdfcpu boxes add    description inFile [ outFile ] [flags]
pdfcpu boxes remove boxTypes inFile [ outFile ] [flags]
```

## [Content](/content)

Add, remove, and manage visible or viewer-facing page content.

```
pdfcpu watermark add    string | file description inFile [ outFile ] [flags]
pdfcpu watermark update string | file description inFile [ outFile ] [flags]
pdfcpu watermark remove inFile [ outFile ] [flags]

pdfcpu stamp add    string | file description inFile [ outFile ] [flags]
pdfcpu stamp update string | file description inFile [ outFile ] [flags]
pdfcpu stamp remove inFile [ outFile ] [flags]

pdfcpu annotations list   inFile [flags]
pdfcpu annotations remove inFile [ outFile ] [ objNr | annotId | annotType ]... [flags]

pdfcpu bookmarks list   inFile [flags]
pdfcpu bookmarks export inFile [ outFileJSON ] [flags]
pdfcpu bookmarks import inFile inFileJSON [ outFile ] [flags]
pdfcpu bookmarks remove inFile [ outFile ] [flags]

pdfcpu pagemode list  inFile [flags]
pdfcpu pagemode set   inFile value [ outFile ] [flags]
pdfcpu pagemode reset inFile [ outFile ] [flags]

pdfcpu pagelayout list  inFile [flags]
pdfcpu pagelayout set   inFile value [ outFile ] [flags]
pdfcpu pagelayout reset inFile [ outFile ] [flags]

pdfcpu viewerpref list  inFile [flags]
pdfcpu viewerpref set   inFile ( inFileJSON | JSONstring ) [ outFile ] [flags]
pdfcpu viewerpref reset inFile [ outFile ] [flags]
```

## [Resource](/resource)

Import, generate, list, update, or package reusable PDF resources and metadata.

```
pdfcpu import [ description ] outFile imageFile... [flags]
pdfcpu grid   [ description ] outFile m n inFile | imageFiles... [flags]

pdfcpu fonts list       [flags]
pdfcpu fonts install    fontFiles... [flags]
pdfcpu fonts cheatsheet [ fontNames... ] [flags]

pdfcpu images list    inFile... [flags]
pdfcpu images extract inFile outDir [flags]
pdfcpu images update  inFile imageFile [ outFile ] [ objNr | (pageNr Id) ] [flags]

pdfcpu attachments list    inFile [flags]
pdfcpu attachments add     inFile file [ , desc ]... [flags]
pdfcpu attachments remove  inFile [ file... ] [flags]
pdfcpu attachments extract inFile outDir [ file... ] [flags]

pdfcpu portfolio list    inFile [flags]
pdfcpu portfolio add     inFile file [ , desc ]... [flags]
pdfcpu portfolio remove  inFile [ file... ] [flags]
pdfcpu portfolio extract inFile outDir [ file... ] [flags]

pdfcpu keywords list   inFile [flags]
pdfcpu keywords add    inFile [ outFile ] keyword... [flags]
pdfcpu keywords remove inFile [ outFile ] [ keyword... ] [flags]

pdfcpu properties list   inFile [flags]
pdfcpu properties add    inFile [ outFile ] nameValuePair... [flags]
pdfcpu properties remove inFile [ outFile ] [ name... ] [flags]
```

## [Extract](/extract/extract)

Extract PDF resources and structural data into an output directory.

```
pdfcpu extract inFile outDir [flags]
```

## [Form](/form/form)

Inspect, fill, reset, lock, and export PDF forms.

```
pdfcpu form list      inFile... [flags]
pdfcpu form remove    inFile [ outFile ] < fieldID | fieldName >... [flags]
pdfcpu form lock      inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form unlock    inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form reset     inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form export    inFile [ outFileJSON ] [flags]
pdfcpu form fill      inFile inFileJSON [ outFile ] [flags]
pdfcpu form multifill inFile inFileData outDir [ outFile ] [flags]
```

## [Security](/encrypt/security)

Encrypt PDFs, decrypt PDFs, change passwords, and manage permissions.

```
pdfcpu encrypt     inFile [ outFile ] [flags]
pdfcpu decrypt     inFile [ outFile ] [flags]
pdfcpu changeopw   inFile opwOld opwNew [ outFile ] [flags]
pdfcpu changeupw   inFile upwOld upwNew [ outFile ] [flags]
pdfcpu permissions list inFile... [flags]
pdfcpu permissions set  inFile [ outFile ] [flags]
```

## [Trust](/core/trust)

Manage certificates and digital signatures.

```
pdfcpu certificates list    [flags]
pdfcpu certificates inspect inFile [flags]
pdfcpu certificates import  inFile... [flags]
pdfcpu certificates reset   [flags]

pdfcpu signatures remove   inFile [ outFile ] [flags]
pdfcpu signatures validate inFile [flags]
```

## [Config](/config/config)

Manage the pdfcpu configuration.

```
pdfcpu config list  [flags]
pdfcpu config reset [flags]
```

## [Paper](/paper)

Print supported paper sizes.

```
pdfcpu paper [flags]
```

## [Page Selection](/getting_started/page_selection/)

Print the definition of the `--pages` flag.

```
pdfcpu selectedpages [flags]
```

## Shell Completion

Generate completion scripts for your shell.

```
pdfcpu completion [bash|zsh|fish|powershell] [flags]
```

## Version

Print the installed pdfcpu version.

```
pdfcpu version [flags]
```
