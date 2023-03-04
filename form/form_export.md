---
layout: default
---

# Export form to JSON

This command creates a JSON file containing a PDF form structure with optional data.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu form export inFile [outFileJSON]
```
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
pdfcpu export english.pdf
writing out.json...

cat out.json

{
	"header": {
		"source": "english.pdf",
		"version": "pdfcpu v0.4.0",
		"creation": "2023-03-04 20:22:17 CET",
		"producer": "pdfcpu v0.4.0"
	},
	"forms": [
		{
			"textfield": [
				{
					"id": "firstName1",
					"default": "Joe",
					"value": "Jackie",
					"multiline": false,
					"locked": false
				},
				{
					"id": "note1",
					"value": "This is a sample text.\nThis is the next line.",
					"multiline": true,
					"locked": false
				}
			],
			"datefield": [
				{
					"id": "dob1",
					"format": "dd.mm.yyyy",
					"default": "01.01.2000",
					"value": "31.12.1999",
					"locked": true
				}
			],
			"checkbox": [
				{
					"id": "cb11",
					"default": false,
					"value": true,
					"locked": false
				},
			],
			"radiobuttongroup": [
				{
					"id": "gender1",
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
					"id": "city12",
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
					"id": "city11",
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

