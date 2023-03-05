---
layout: default
---

# List form fields

This command prints a list of all form fields for a PDF form.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu form list inFile...
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
| inFile       | PDF input file containing a form      | yes

<br>

## Examples

List all form fields of english.pdf:

```
pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ firstName1 │ Joe              │ Jackie                   │
     Textfield │ lastName1  │ Doeby            │ Doe                      │
     Datefield │ dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ cb11       │                  │ Yes                      │
     Textfield │ note1      │                  │ This is a sample text.\n │
```

<br>

List all form fields of person.pdf.

The * indicates locked fields aka write-protected or read-only:

```
pdfcpu form list person.pdf

person.pdf
Pg L Field     │ Name        │ Value   │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ firstName   │         │
     Textfield │ lastName    │         │
     Datefield │ dob         │         │
     CheckBox  │ dobVerified │         │
     RadioBGr. │ gender      │         │ female,male,non-binary
     Textfield │ country     │         │
   * Textfield │ planet      │ Earth   │
     Textfield │ occup       │         │
     ComboBox  │ status      │ unknown │ alive,deceased,imprisoned,killed,unknown
     ComboBox  │ license     │         │ CC BY 4.0,CC BY-SA 2.0,Public Domain,unknown
```