---
layout: default
---

# Export Bookmarks

This command exports existing bookmarks to a JSON dataset.<br>
The resulting JSON structure is also used by `pdfcpu bookmarks import`.


Have a look at some [examples](#examples).

## Usage

```
pdfcpu bookmarks export inFile [outFileJSON]
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)       | user password   |
| [opw](../getting_started/common_flags.md)       | owner password  |

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:------
| inFile       | PDF input file      | yes      |
| outFileJSON  | JSON output file    | no       | out.json

<br>

## Examples

```sh
$ pdfcpu bookm export bookmarkTree.pdf
writing out.json...
$ cat out.json
{
	"header": {
		"source": "bookmarkTree.pdf",
		"version": "pdfcpu v0.5.0 dev",
		"creation": "2023-08-19 12:53:28 CEST",
		"title": "The Center of Why?\"",
		"author": "Alan Kay",
		"creator": "Acrobat PDFMaker 5.0 for Word",
		"producer": "pdfcpu v0.4.1 dev",
		"subject": "2004 Kyoto Prize Commorative Lecture"
	},
	"bookmarks": [
		{
			"title": "Page 1: Level 1",
			"page": 1,
			"color": {
				"R": 0,
				"G": 1,
				"B": 0
			},
			"kids": [
				{
					"title": "Page 2: Level 1.1",
					"page": 2
				},
				{
					"title": "Page 3: Level 1.2",
					"page": 3,
					"kids": [
						{
							"title": "Page 4: Level 1.2.1",
							"page": 4
						}
					]
				}
			]
		},
		{
			"title": "Page 5: Level 2",
			"page": 5,
			"color": {
				"R": 0,
				"G": 0,
				"B": 1
			},
			"kids": [
				{
					"title": "Page 6: Level 2.1",
					"page": 6
				},
				{
					"title": "Page 7: Level 2.2",
					"page": 7
				},
				{
					"title": "Page 8: Level 2.3",
					"page": 8
				}
			]
		}
	]
}
```