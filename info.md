---
layout: default
---

# Info

Print information about a PDF file and its attachments.

## Usage

```
pdfcpu info [-pages selectedPages] [-j(son)] inFile...
```

<br>

### Flags

| name                                      | description         | required
|:----------------------------------------  |:--------------------|:--------
| [p(ages)](getting_started/page_selection) | page selection      | no
| j(son)                                    | produce JSON output | no

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input files     | yes

## Example

```sh
$ pdfcpu info -u cm test.pdf
              Source: test.pdf
         PDF version: 1.7
          Page count: 1
           Page size: 21.00 x 29.70 cm
---------------------------------------------
               Title:
              Author:
             Subject:
        PDF Producer: pdfcpu v0.6.0
     Content creator:
       Creation date: D:20231223010752+02'00'
   Modification date: D:20231223010752+02'00'
           Page mode: UseThumbs
         Page Layout: SinglePage
        Viewer Prefs: HideToolbar = true
                      HideMenubar = true
                      FitWindow = true
                      CenterWindow = true
                      NonFullScreenPageMode = UseNone
            Keywords: key1
                      key2
          Properties: name1 = val1
                      name2 = val2
---------------------------------------------
              Tagged: No
              Hybrid: No
          Linearized: No
  Using XRef streams: Yes
Using object streams: Yes
         Watermarked: No
          Thumbnails: No
            Acroform: No
            Outlines: Yes
               Names: Yes
---------------------------------------------
           Encrypted: No
         Permissions: Full access
```
<br>

Use the *pages* flag to include page boundaries for selected pages in your desired display unit:<br><br>
w  ... width<br>
h  ... height<br>
ar ... aspect ratio

```sh
$ pdfcpu info -u po -pages 1,2 test.pdf
pages: 1,2
              Source: test.pdf
         PDF version: 1.2
          Page count: 2
Page 1: rot=+0 orientation:portrait
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71  = CropBox, TrimBox, BleedBox, ArtBox
Page 2: rot=+0 orientation:portrait
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71  = CropBox, TrimBox, BleedBox, ArtBox
---------------------------------------------
               Title:
              Author:
             Subject:
        PDF Producer: DOC1/EMFE v4.4M0p2286 + SCR 57461
     Content creator:
       Creation date: D:20150122062117
   Modification date:
---------------------------------------------
              Tagged: No
              Hybrid: No
          Linearized: No
  Using XRef streams: No
Using object streams: No
         Watermarked: No
          Thumbnails: No
            Acroform: No
            Outlines: Yes
               Names: Yes
---------------------------------------------
           Encrypted: No
         Permissions: Full access
```

<br>

Output a JSON data set:

```
$ pdfcpu info -json test.pdf
{
	"header": {
		"version": "pdfcpu v0.5.0 dev",
		"creation": "2023-08-20 00:24:45 CEST"
	},
	"Infos": [
		{
			"source": "test.pdf",
			"version": 1.7,
			"pages": 1,
			"title": "",
			"author": "",
			"subject": "",
			"producer": "pdfcpu v0.3.6 dev",
			"creator": "",
			"creationDate": "D:20201103224901+01'00'",
			"modificationDate": "D:20201103224901+01'00'",
			"keywords": [],
			"properties": {},
			"tagged": false,
			"hybrid": false,
			"linearized": false,
			"usingXRefStreams": true,
			"usingObjectStreams": true,
			"watermarked": false,
			"thumbnails": false,
			"form": false,
			"signatures": false,
			"appendOnly": false,
			"bookmarks": false,
			"names": false,
			"encrypted": false,
			"permissions": 0,
		}
	]
}
```
