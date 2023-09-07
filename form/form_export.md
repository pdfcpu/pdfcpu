---
layout: default
---

# Export form to JSON

This command creates a JSON file containing a PDF form structure with optional data.

The resulting JSON payload contains the single element array `forms` serving as a starting point for form filling.

The content of this element contains all form fields grouped by field type:
* text fields
* date fields
* check boxes
* radio button groups
* combo boxes
* list boxes


Have a look at some [examples](#examples).

## Usage

```
pdfcpu form export inFile [outFileJSON]
```
<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file containing form      | yes
| outFileJSON  | JSON output file    | no

<br>

## Examples

Export a form created with pdfcpu to JSON:

```
$ pdfcpu form export english.pdf
writing out.json...

$ cat out.json

{
	"header": {
		"source": "english.pdf",
		"version": "pdfcpu v0.4.1",
		"creation": "2023-03-04 20:22:17 CET",
		"producer": "pdfcpu v0.4.1"
	},
	"forms": [
		{
			"textfield": [
				{
					"page": 1,
					"id": "30",
					"name": "firstName1",
					"default": "Joe",
					"value": "Jackie",
					"multiline": false,
					"locked": false
				},
				{
					"page": 1,
					"id": "31",
					"name": "note1",
					"value": "This is a sample text.\nThis is the next line.",
					"multiline": true,
					"locked": false
				}
			],
			"datefield": [
				{
					"page": 1,
					"id": "33",
					"name": "dob1",
					"format": "dd.mm.yyyy",
					"default": "01.01.2000",
					"value": "31.12.1999",
					"locked": true
				}
			],
			"checkbox": [
				{
					"page": 1,
					"id": "34",
					"name": "cb11",
					"default": false,
					"value": true,
					"locked": false
				},
			],
			"radiobuttongroup": [
				{
					"page": 1,
					"id": "35",
					"name": "gender1",
					"options": [
						"female",
						"male",
						"non-binary"
					],
					"default": "male",
					"value": "non-binary",
					"locked": false
				}
			],
			"combobox": [
				{
					"page": 1,
					"id": "36",
					"name": "city12",
					"editable": false,
					"options": [
						"London",
						"San Francisco",
						"Sidney"
					],
					"default": "San Francisco",
					"value": "Sidney",
					"locked": false
				}
			],
			"listbox": [
				{
					"page": 1,
					"id": "37",
					"name": "city11",
					"multi": true,
					"options": [
						"San Francisco",
						"São Paulo",
						"Vienna"
					],
					"defaults": [
						"Vienna",
						"São Paulo"
					],
					"values": [
						"San Francisco",
						"Vienna"
					],
					"locked": false
				}
			]
		}
	]
}
```

