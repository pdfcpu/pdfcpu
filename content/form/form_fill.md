---
layout: default
title: "Fill form via JSON"
---

# Fill form via JSON

This command fills form fields with data via JSON.

1. Export your form into JSON using `pdfcpu form export`.

2. Edit `value` (or `values` where appropriate) for all form fields you want to fill in the exported file.

3. In addition to modifying `value(s)` you may change the `locked` status for fields.

4. Remove all fields which shall remain untouched.

5. Run `pdfcpu form fill`. This will process the attributes `value` and `locked` only.

## Note
pdfcpu generates all appearance streams for form fields, but if you do have form field display
issues then the following configuration option may help:
`needAppearances: true`

When filling radio button groups you may provide either the displayed choice or its array index. Export the form first if
you need to see the exact choices. Choice names containing spaces or `#` characters are supported, including in forms
created with older pdfcpu releases.

When pdfcpu generates field appearances, it uses the font encoding stored in the form. If the form does not define a
usable glyph for a value, filling reports an error instead of silently displaying the wrong character.


Have a look at some [examples](#examples). 

## Usage

```
pdfcpu form fill inFile inFileJSON [ outFile ] [flags]
```
<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file containing form, use `-` to read from stdin      | yes
| inFileJSON   | JSON input file with form data    | yes
| outFile      | PDF output file, use `-` to write to stdout      | no

<br>

## Examples

Use an exported JSON file to fill `firstName` and `dob` and make `dob` read-only:

Field identification may be processed via "id" or "name".

We edit the JSON file:
```
{
	"header": {
		"source": "english.pdf",
		"version": "pdfcpu v0.4.1",
		"creation": "2023-04-04 20:22:17 CET",
		"producer": "pdfcpu v0.4.1"
	},
	"forms": [
		{
			"textfield": [
				{
					"name": "firstName",
					"value": "Horst",
					"locked": false
				}
			],
			"datefield": [
				{
					"name": "dob",
					"value": "31.12.1999",
					"locked": true
				}
			]
		}
	]
}
```

We trigger (a dry run for) form filling and write the filled form to `tmp.pdf`:
```
$ pdfcpu form fill english.pdf english.json tmp.pdf
```

We check the result by exporting the form out of `tmp.pdf`:

```
$ pdfcpu form export tmp.pdf tmp.json
writing tmp.json...
```

* We inspect tmp.json and are satisfied with the result.
* We open tmp.pdf in Adobe Reader and are satisfied with the result.
* We fill the original form:

```
$ pdfcpu form fill english.pdf english.json
```

<br>

Fill a streamed form with a local JSON data file:

```sh
$ aws s3 cp s3://acme-forms/application.pdf - \
   | pdfcpu form fill - application.json - \
   | aws s3 cp - s3://acme-forms/application-filled.pdf
```
