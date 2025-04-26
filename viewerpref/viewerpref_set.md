---
layout: default
---

# Set Viewer Preferences

This command configures how a PDF is displayed on the screen and shall be printed.

Define whether you want to display a toolbar, a menubar or if you want your document to be centered among others.

Define a default print page range, your paper handling options and other parameters for printing.

Have a look at some [examples](#examples).





## Usage

```
pdfcpu viewerpref set inFile (inFileJSON | JSONstring)
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file                             | yes
| inFileJSON   | JSON input file or                         | yes
| JSONstring   | JSON string containing viewing preferences | yes

<br>

### Preference Parameters

| name         | description | default
|:----------------|:---------|:----------------------------------------
| HideToolbar     | Hide tool bars when the document is active            | false
| HideMenubar     | Hide the menu bar when the document is active         | false
| HideWindowUI    | Hide user interface elements in the document’s window | false
| FitWindow       | Resize the document’s window to fit the size of the first displayed page | false
| CenterWindow    | Position the document’s window in the centre of the screen               | false
| DisplayDocTitle | Display the document title | false
| NonFullScreenPageMode | How to display the document on exiting full-screen mode | UseNone
| Direction       | The predominant logical content order for text | L2R
| ViewArea        | Viewing Page boundary for screen               | CropBox
| ViewClip        | Clipping page boundary for screen              | CropBox
| PrintArea       | Rendering Page boundary for printing           | CropBox
| PrintClip       | Clipping page boundary for printing            | CropBox
| Duplex          | Paper handling option                          | -
| PickTrayByPDFSize | Whether the PDF page size shall be used to select the input paper tray | -
| PrintPageRange  | The page numbers used to initialize the print dialogue box when the file is printed (since PDF 1.7). The array shall contain an even number of integers to be interpreted in pairs, with each pair specifying the first and last pages in a sub-range of pages to be printed. The first page of the PDF file shall be denoted by 1. | -
| NumCopies       | The number of copies that shall be printed when the print dialog is opened for this file (since PDF 1.7). | -
| Enforce         | Array of names of Viewer preference settings that shall be enforced by PDF processors and that shall not be overridden by subsequent selections in the application user interface (since PDF 2.0) - possible value: PrintScaling | -



<br>

#### DisplayDocTitle

| value | description
|:------|:------------
| true   | The window’s title bar should display the document title taken from the dc:title element of the XMP metadata stream 
| false  | The title bar should display the name of the PDF file containing the document 

<br>

#### NonFullScreenPageMode

| value       | description
|:------------|:------------
| UseNone     | Neither document outline nor thumbnail images visible (=default)
| UseOutlines | Document outline visible
| UseThumbs   | Thumbnail images visible
| UseOC       | Optional content group panel visible

<br>

#### Direction

| value | description
|:------|:------------
|L2R    | Left to right
|R2L    | Right to left (including vertical writing systems, such as Chinese, Japanese, and Korean)

<br>

#### View/Print Area/Clip

Since PDF 1.4, deprecated as of PDF 2.0

Values: The PDF [page boundaries (boxes)](../getting_started/box.md)


| parameter     | description
|:------    |:------------
| ViewArea  | The name of the page boundary representing the area of a page that shall be displayed when viewing the document on the screen.
| ViewClip  | The name of the page boundary to which the contents of a page shall be clipped when viewing the document on the screen.
| PrintArea | The name of the page boundary representing the area of a page that shall be rendered when printing the document.
| PrintClip  | The name of the page boundary to which the contents of a page shall be clipped when printing the document.
                                    
<br>

#### Duplex

The paper handling option that shall be used when printing the file from the print dialogue (since PDF 1.7)

| value | description
|:--------------------|:------------                        
| Simplex             | Print single-sided
| DuplexFlipShortEdge | Duplex and flip on the short edge of the sheet
| DuplexFlipLongEdge  | Duplex and flip on the long edge of the sheet

<br>

## Examples

Generally the viewer preferences are set via JSON.

Eg. Set viewer preferences via JSON string (case agnostic):

```sh
$ pdfcpu viewerpref set test.pdf "{\"HideMenuBar\": true, \"CenterWindow\": true}"

```

<br>

Set printer preferences (which are part of the viewer preferences) via JSON string (case agnostic):

```sh
$ pdfcpu viewerpref set test.pdf "{\"duplex\": \"duplexFlipShortEdge\", \"printPageRange\": [1, 4, 10, 12], \"NumCopies\": 3}"

```

<br>

Set viewer preferences via JSON file:

```sh
$ cat viewerpref.json
{
    "viewerPreferences": {
        "HideToolBar": true,
        "HideMenuBar": false,
        "HideWindowUI": false,
        "FitWindow": true,
        "CenterWindow": true,
        "DisplayDocTitle": true,
        "NonFullScreenPageMode": "UseThumbs",
        "Direction": "R2L",
        "Duplex": "Simplex",
        "PickTrayByPDFSize": false,
        "PrintPageRange": [
            1, 4,
            10, 20
        ],
        "NumCopies": 3,
        "Enforce": [
            "PrintScaling"
        ]
    }
}

$ pdfcpu viewerpref set test.pdf viewerpref.json

```
