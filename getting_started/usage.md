---
layout: default
---

# Usage

Use `pdfcpu` for a rundown of all commands:

```
$ pdfcpu
pdfcpu is a tool for PDF manipulation written in Go.

Usage:

   pdfcpu command [arguments]

Available Commands:
  annotations   List, remove page annotations
  attachments   List, add, remove, extract embedded file attachments
  booklet       Arrange pages onto larger sheets of paper to make a booklet or zine
  bookmarks     List, import, export, remove bookmarks
  boxes         List, add, remove page boundaries for selected pages
  certificates  List, inspect, import, reset certificates
  changeopw     Change owner password
  changeupw     Change user password
  collect       Create custom sequence of selected pages
  completion    Generate shell completion script
  config        List, reset configuration
  create        Create PDF content including forms via JSON
  crop          Set cropbox for selected pages
  cut           Custom cut pages horizontally or vertically
  decrypt       Remove password protection
  encrypt       Set password protection
  extract       Extract images, fonts, content, pages or metadata
  fonts         Install, list supported fonts, create cheat sheets
  form          List, remove fields, lock, unlock, reset, export, fill form via JSON or CSV
  grid          Rearrange pages or images for enhanced browsing experience
  help          Help about any command
  images        List, extract, update images
  import        Import/convert images to PDF
  info          Print file info
  keywords      List, add, remove keywords
  merge         Concatenate PDFs
  ndown         Cut selected page into n pages symmetrically
  nup           Rearrange pages or images for reduced number of pages
  optimize      Optimize PDF by getting rid of redundant page resources
  pagelayout    List, set, reset page layout for opened document
  pagemode      List, set, reset page mode for opened document
  pages         Insert, remove selected pages
  paper         Print list of supported paper sizes
  permissions   List, set user access permissions
  portfolio     List, add, remove, extract portfolio entries
  poster        Create poster using paper size
  properties    List, add, remove document properties
  resize        Scale selected pages
  rotate        Rotate selected pages
  selectedpages Print definition of the -pages flag
  signatures    Remove, validate signatures
  split         Split up inFile by span or bookmark
  stamp         Add, remove, update text, image or PDF stamps for selected pages
  trim          Create trimmed version of selected pages
  validate      Validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic PDF 2.0 validation
  version       Print version
  viewerpref    List, set, reset viewer preferences
  watermark     Add, remove, update watermarks
  zoom          Zoom in/out of selected pages

Flags:
  -c, --conf string     set or disable config dir: $path | disable
  -h, --help            help for pdfcpu
  -o, --offline         disable http traffic
  -q, --quiet           disable output
  -v, --verbose count   Increase verbosity. Use -v or -vv.

Use "pdfcpu [command] --help" for more information about a command.
```

<br>

## [Core Commands](../core/core.md)

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

## [Stamps](../core/stamp.md)

Manage your stamps:
```
pdfcpu stamp add string | file description inFile [ outFile ] [flags]
pdfcpu stamp update string | file description inFile [ outFile ] [flags]
pdfcpu stamp remove inFile [ outFile ] [flags]
```

<br>

## [Watermarks](../core/watermark.md)

Manage your watermarks:
```
pdfcpu watermark add string | file description inFile [ outFile ] [flags]
pdfcpu watermark update string | file description inFile [ outFile ] [flags]
pdfcpu watermark remove inFile [ outFile ] [flags]
```

<br>


## [Forms](../form/form.md)

Manage your PDF forms:

```
pdfcpu form list   inFile... [flags]
pdfcpu form remove inFile [ outFilen] < fieldID | fieldName >... [flags]
pdfcpu form lock   inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form unlock inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form reset  inFile [ outFile ] [ fieldID | fieldName ]... [flags]
pdfcpu form export inFile [ outFileJSON ] [flags]
pdfcpu form fill   inFile inFileJSON [ outFile ] [flags]
```
<br>

## [Fonts](../fonts/fonts.md)

Manage core fonts and your user fonts:

```
pdfcpu fonts list [flags]
pdfcpu fonts install fontFiles... [flags]
pdfcpu fonts cheatsheet fontFiles... [flags]
```

<br>

## [Generate Commands](../generate/generate.md)

```
pdfcpu booklet [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu create  inFileJSON [ inFile ] outFile [flags]
pdfcpu cut     description inFile outDir [ outFile ] [flags]
pdfcpu grid    [ description ] outFile m n inFile | imageFiles... [flags]
pdfcpu import  [ description ] outFile imageFile... [flags]
pdfcpu ndown   [ description ] n inFile outDir [ outFile ] [flags]
pdfcpu nup     [ description ] outFile n inFile | imageFiles... [flags]
pdfcpu poster  description inFile outDir [ outFile] [flags]
```

<br>

## [Pages](../pages/pages.md)

Insert or remove pages:
```
pdfcpu pages insert [ description ] inFile [ outFile ] [flags]
pdfcpu pages remove inFile [ outFile ] [flags]
```

<br>

## [Extract](../extract/extract.md)

Extract components and resources like fonts and images:
```
pdfcpu extract inFile outDir [flags]
```

<br>

## [Attachments](../attach/attach.md)

Manage your attachments:
```
pdfcpu attachments list inFile [flags]
pdfcpu attachments add inFile file... [flags]
pdfcpu attachments remove inFile [ file... ] [flags]
pdfcpu attachments extract inFile outDir [ file... ] [flags]
```

<br>

## [Portfolio](../portfolio/portfolio.md)

Manage your portfolios:
```
pdfcpu portfolio list inFile [flags]
pdfcpu portfolio add inFile file... [flags]
pdfcpu portfolio remove inFile [ file... ] [flags]
pdfcpu portfolio extract inFile outDir [ file... ] [flags]
```

<br>

## [Annotations](../annot/annot.md)

Manage your annotations:
```
pdfcpu annotations list inFile [flags]
pdfcpu annotations remove inFile [ outFile ] [ objNr | annotId | annotType]... [flags]
```

<br>

## [Images](../images/images.md)

Manage your images:
```
pdfcpu images list inFile... [flags]
pdfcpu images extract inFile outDir [flags]
pdfcpu images update inFile imageFile [ outFile ] [ objNr | (pageNr Id) ] [flags]
```

<br>

## [Encryption](../encrypt/encrypt.md)

Secure your PDFs:
```
pdfcpu encrypt     inFile [ outFile ] [flags]
pdfcpu decrypt     inFile [ outFile ] [flags]
pdfcpu changeopw   inFile opwOld opwNew [flags]
pdfcpu changeupw   inFile upwOld upwNew [flags]
pdfcpu permissions list inFile... [flags]
pdfcpu permissions set inFile [flags]
```

<br>

## [Print Supported Papersizes](../paper.md)

```
pdfcpu paper [flags]
```
<br>

## [Keywords](../keywords/keywords.md)

Manage your keywords for searching:
```
pdfcpu keywords list inFile [flags]
pdfcpu keywords add inFile keyword... [flags]
pdfcpu keywords remove inFile [ keyword... ] [flags]
```

<br>

## [Properties](../properties/properties.md)

Manage your document properties:
```
pdfcpu properties list inFile [flags]
pdfcpu properties add inFile nameValuePair... [flags]
pdfcpu properties remove inFile [ name... ] [flags]
```

<br>

## [Page Layout](../pagelayout/pagelayout.md)

Manage the page layout for your opened document:
```
pdfcpu pagelayout list inFile [flags]
pdfcpu pagelayout set inFile value [flags]
pdfcpu pagelayout reset inFile [flags]
```

<br>

## [Page Mode](../pagemode/pagemode.md)

Manage the page mode for your opened document:
```
pdfcpu pagemode list inFile [flags]
pdfcpu pagemode set inFile value [flags]
pdfcpu pagemode reset inFile [flags]
```

<br>

## [Signatures]()

Manage digital signatures:
```
pdfcpu signatures remove inFile [ outFile ] [flags]
pdfcpu signatures validate inFile [flags]
```

<br>

## [Viewer Preferences](../viewerpref/viewerpref.md)

Manage the viewer preferences for your opened document:
```
pdfcpu viewerpref list inFile [flags]
pdfcpu viewerpref set inFile ( inFileJSON | JSONstring ) [flags]
pdfcpu viewerpref reset inFile [flags]
```

<br>

## [Bookmarks](../bookmarks/bookmarks.md)

Manage your bookmarks:
```
pdfcpu bookmarks list inFile [flags]
pdfcpu bookmarks import inFile inFileJSON [ outFile ] [flags]
pdfcpu bookmarks export inFile [ outFileJSON ] [flags]
pdfcpu bookmarks remove inFile [ outFile ] [flags]
```

<br>

## [Boxes](../boxes/boxes.md)

Manage your page boundaries:
```
pdfcpu boxes list [ boxTypes ] inFile [flags]
pdfcpu boxes add description inFile [ outFile ] [flags]
pdfcpu boxes remove boxTypes inFile [ outFile ] [flags]
```

<br>

## [Config](../boxes/boxes.md)

Manage your configuration:
```
pdfcpu config list [flags]
pdfcpu config reset [flags]
```

<br>

## [Info](../info.md)

Print file details:
```
pdfcpu info inFile... [flags]
```

<br>

## [Print definition of the -pages flag](../getting_started/page_selection.md)

```
pdfcpu selectedpages [flags]
```

<br>

## Print Version

```
pdfcpu version [flags]
```

