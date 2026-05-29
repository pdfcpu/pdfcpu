---
layout: default
title: "List Viewer Preferences"
---

# List Viewer Preferences

This command outputs a list of any configured viewer preferences.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu viewerpref list inFile [flags]
```

<br>

### Flags

| name   | description                            | default | required
|:-------|:---------------------------------------|:--------|:--
| a(ll)  | output all, including default values   | no      | no
| json   | output JSON                            | no      | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name    | description         | required
|:--------|:--------------------|:--------------------------
| inFile  | PDF input file, use `-` to read from stdin      | yes



<br>

## Examples

Display all non-default viewer preferences:

```sh
$ pdfcpu viewerpref list test.pdf
Viewer preferences:
   DisplayDocTitle = true
```

<br>

Display all viewer preferences:
```sh
$ pdfcpu viewerpref list test.pdf --all
Viewer preferences:
   HideToolbar = false
   HideMenubar = false
   HideWindowUI = false
   FitWindow = false
   CenterWindow = false
   DisplayDocTitle = true
   NonFullScreenPageMode = UseNone
   Direction = L2R
   PrintScaling = AppDefault
   NumCopies = 1
```

<br>

Display all non-default viewer preferences using JSON:
```sh
$ pdfcpu viewerpref list --json test.pdf
{
	"header": {
		"version": "pdfcpu vX.Y.Z",
		"creation": "YYYY-MM-DD HH:MM:SS TZ"
	},
	"viewerPreferences": {
		"displayDocTitle": true
	}
}
```

<br>

Display all viewer preferences using JSON:
```sh
$ pdfcpu viewerpref list --all --json test.pdf
{
	"header": {
		"version": "pdfcpu vX.Y.Z",
		"creation": "YYYY-MM-DD HH:MM:SS TZ"
	},
	"viewerPreferences": {
		"hideToolbar": false,
		"hideMenubar": false,
		"hideWindowUI": false,
		"fitWindow": false,
		"centerWindow": false,
		"displayDocTitle": true,
		"nonFullScreenPageMode": "UseNone",
		"direction": "L2R",
		"printScaling": "AppDefault",
		"numCopies": 1
	}
}
```

<br>

List viewer preferences for a streamed PDF:

```sh
$ aws s3 cp s3://acme-print/catalog.pdf - \
   | pdfcpu viewerpref list -
```
