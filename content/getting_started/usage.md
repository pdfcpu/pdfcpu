---
layout: default
title: "Usage"
---

# Usage

Explore the available commands and their usage.

Use:

    pdfcpu [command] --help

for detailed information about a specific command.

---
<br>

Commands are grouped by functionality:

## [Core Commands](/core/core)

The basic processing features:
```
pdfcpu collect  inFile [ outFile ] [flags]
pdfcpu crop     description inFile [ outFile ] [flags]
pdfcpu merge    outFile inFile... [flags]
pdfcpu optimize inFile [ outFile ] [flags]
pdfcpu resize   description inFile [ outFile ] [flags]
pdfcpu rotate   inFile rotation [outFile] [flags]
pdfcpu split    inFile outDir [ span | pageNr... ] [flags]
pdfcpu trim     inFile [outFile] [flags]
pdfcpu validate inFile... [flags]
pdfcpu zoom     description inFile [outFile] [flags]
```

<br>

## [Stamps](/core/stamp)

Manage your stamps:
```
pdfcpu stamp add string | file description inFile [ outFile ] [flags]
pdfcpu stamp update string | file description inFile [ outFile ] [flags]
pdfcpu stamp remove inFile [ outFile ] [flags]
```

<br>

## [Watermarks](/core/watermark)

Manage your watermarks:
```
pdfcpu watermark add string | file description inFile [ outFile ] [flags]
pdfcpu watermark update string | file description inFile [ outFile ] [flags]
pdfcpu watermark remove inFile [ outFile ] [flags]
```

<br>


## [Forms](/form/form)

Manage your PDF forms:

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
<br>

## [Fonts](/fonts/fonts)

Manage core fonts and your user fonts:

```
pdfcpu fonts list [flags]
pdfcpu fonts install fontFiles... [flags]
pdfcpu fonts cheatsheet fontFiles... [flags]
```

<br>

## [Generate Commands](/generate/generate)

```
pdfcpu booklet [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu create  inFileJSON [ inFile ] outFile [flags]
pdfcpu cut     description inFile outDir [ outFile ] [flags]
pdfcpu grid    [ description ] outFile m n inFile | imageFiles... [flags]
pdfcpu import  [description] outFile imageFile... [flags]
pdfcpu ndown   [ description ] n inFile outDir [ outFile ] [flags]
pdfcpu nup     [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu poster  description inFile outDir [ outFile] [flags]
```

<br>

## [Pages](/pages/pages)

Insert or remove pages:
```
pdfcpu pages insert [ description ] inFile [ outFile ] [flags]
pdfcpu pages remove inFile [ outFile ] [flags]
```

<br>

## [Extract](/extract/extract)

Extract components and resources like fonts and images:
```
pdfcpu extract inFile outDir [flags]
```

<br>

## [Attachments](/attach/attach)

Manage your attachments:
```
pdfcpu attachments list inFile [flags]
pdfcpu attachments add inFile file [ , desc ]... [flags]
pdfcpu attachments remove inFile [ file... ] [flags]
pdfcpu attachments extract inFile outDir [ file... ] [flags]
```

<br>

## [Portfolio](/portfolio/portfolio)

Manage your portfolios:
```
pdfcpu portfolio list inFile [flags]
pdfcpu portfolio add inFile file [ , desc ]... [flags]
pdfcpu portfolio remove inFile [ file... ] [flags]
pdfcpu portfolio extract inFile outDir [ file... ] [flags]
```

<br>

## [Annotations](/annot/annot)

Manage your annotations:
```
pdfcpu annotations list inFile [flags]
pdfcpu annotations remove inFile [ outFile ] [ objNr | annotId | annotType]... [flags]
```

<br>

## [Images](/images/images)

Manage your images:
```
pdfcpu images list inFile... [flags]
pdfcpu images extract inFile outDir [flags]
pdfcpu images update inFile imageFile [ outFile ] [ objNr | (pageNr Id) ] [flags]
```

<br>

## [Encryption](/encrypt/encrypt)

Secure your PDFs:
```
pdfcpu encrypt     inFile [ outFile ] [flags]
pdfcpu decrypt     inFile [ outFile ] [flags]
pdfcpu changeopw   inFile opwOld opwNew [ outFile ] [flags]
pdfcpu changeupw   inFile upwOld upwNew [ outFile ] [flags]
pdfcpu permissions list inFile... [flags]
pdfcpu permissions set inFile [ outFile ] [flags]
```

<br>

## [Certificates](/core/certs)

Manage certificates:
```
pdfcpu certificates list [flags]
pdfcpu certificates inspect inFile [flags]
pdfcpu certificates import inFile... [flags]
pdfcpu certificates reset [flags]
```

<br>

## [Print Supported Papersizes](/paper)

```
pdfcpu paper [flags]
```
<br>

## [Keywords](/keywords/keywords)

Manage your keywords for searching:
```
pdfcpu keywords list inFile [flags]
pdfcpu keywords add inFile [ outFile ] keyword... [flags]
pdfcpu keywords remove inFile [ outFile ] [ keyword... ] [flags]
```

<br>

## [Properties](/properties/properties)

Manage your document properties:
```
pdfcpu properties list inFile [flags]
pdfcpu properties add inFile [ outFile ] nameValuePair... [flags]
pdfcpu properties remove inFile [ outFile ] [ name... ] [flags]
```

<br>

## [Page Layout](/pagelayout/pagelayout)

Manage the page layout for your opened document:
```
pdfcpu pagelayout list inFile [flags]
pdfcpu pagelayout set inFile value [ outFile ] [flags]
pdfcpu pagelayout reset inFile [ outFile ] [flags]
```

<br>

## [Page Mode](/pagemode/pagemode)

Manage the page mode for your opened document:
```
pdfcpu pagemode list inFile [flags]
pdfcpu pagemode set inFile value [ outFile ] [flags]
pdfcpu pagemode reset inFile [ outFile ] [flags]
```

<br>

## [Signatures](/core/sign)

Manage digital signatures and validate signature integrity:
```
pdfcpu signatures remove inFile [ outFile ] [flags]
pdfcpu signatures validate inFile [flags]
```

<br>

## [Viewer Preferences](/viewerpref/viewerpref)

Manage the viewer preferences for your opened document:
```
pdfcpu viewerpref list inFile [flags]
pdfcpu viewerpref set inFile ( inFileJSON | JSONstring ) [ outFile ] [flags]
pdfcpu viewerpref reset inFile [ outFile ] [flags]
```

<br>

## [Bookmarks](/bookmarks/bookmarks)

Manage your bookmarks:
```
pdfcpu bookmarks list inFile [flags]
pdfcpu bookmarks import inFile inFileJSON [ outFile ] [flags]
pdfcpu bookmarks export inFile [ outFileJSON ] [flags]
pdfcpu bookmarks remove inFile [ outFile ] [flags]
```

<br>

## [Boxes](/boxes/boxes)

Manage your page boundaries:
```
pdfcpu boxes list [ boxTypes ] inFile [flags]
pdfcpu boxes add description inFile [ outFile ] [flags]
pdfcpu boxes remove boxTypes inFile [ outFile ] [flags]
```

<br>

## [Config](/config/config)

Manage your configuration:
```
pdfcpu config list [flags]
pdfcpu config reset [flags]
```

<br>

## [Info](/info)

Print file details:
```
pdfcpu info inFile... [flags]
```

<br>

## [Print definition of the -pages flag](/getting_started/page_selection)

```
pdfcpu selectedpages [flags]
```

<br>

## Print Version

```
pdfcpu version [flags]
```
