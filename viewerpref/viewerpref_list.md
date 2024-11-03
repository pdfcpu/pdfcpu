---
layout: default
---

# List Viewer Preferences

This command outputs a list of any configured viewer preferences.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu viewerpref list [-a(ll)] [-j(son)] inFile
```

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

| name    | description         | required
|:--------|:--------------------|:--------------------------
| all     | output all (including default values)      | no
| json    | output JSON                                | no
| inFile  | PDF input file                             | yes



<br>

## Examples

Display all non default viewer preferences:

```sh
$ pdfcpu viewerpref list test.pdf
Viewer preferences:
   DisplayDocTitle = true
```

<br>

Display all viewer preferences:
```sh
$ pdfcpu viewerpref list -all test.pdf
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

Display all non default viewer preferences using JSON:
```sh
$ pdfcpu viewerpref list -json test.pdf
{
	"header": {
		"version": "pdfcpu v0.6.0 dev",
		"creation": "2023-12-05 14:23:56 CET"
	},
	"viewerPreferences": {
		"displayDocTitle": true
	}
}
```

<br>

Display all viewer preferences using JSON:
```sh
$ pdfcpu viewerpref list -all -json test.pdf
{
	"header": {
		"version": "pdfcpu v0.6.0 dev",
		"creation": "2023-12-05 14:24:04 CET"
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

