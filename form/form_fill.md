---
layout: default
---

# Fill form via JSON

This command fills form fields with data via JSON.

1. Export your form into JSON using `pdfcpu form export`.

2. Edit `value` (or `values` where appropriate) for all form fields you want to fill in the exported file.

3. In addition to modifying `value(s)` you may change the `locked` status for fields.

3. Remove all fields which shall remain untouched.

4. Run `pdfcpu form fill`. This will process the attributes `value` and `locked` only.

Have a look at some [examples](#examples). 

## Usage

```
pdfcpu form fill inFile inFileJSON [outFile]
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
| inFileJSON   | JSON input file with form data    | yes
| outFile      | PDF output file for dry runs      | no

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
			],
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
$ pdfcpu export tmp.pdf tmp.json
writing tmp.json...
```

* We inspect tmp.json and are satisfied with the result.
* We open tmp.pdf in Adobe Reader and are satisfied with the result.
* We fill the original form:

```
$ pdfcpu form fill english.pdf english.json
```
