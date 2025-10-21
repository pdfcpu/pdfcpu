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

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file containing a form      | yes

<br>

Use the following configuration parameter in order to tweak verbose output:
```
# limit form field content for display purposes when using pdfcpu form list.
# if > 0 affects the columns AltName, Default and Value.
FormFieldListMaxColWidth: 0
```

## Examples

List all form fields of english.pdf:

```
$ pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
     Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
     Datefield │ 34 | dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ 36 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 17 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 38 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 40 | cb11       │                  │ Yes                      │
     Textfield │ 41 | note1      │                  │ This is a sample text.\n │
```

<br>

List all form fields of person.pdf.

The * indicates locked fields aka write-protected or read-only:

```
$ pdfcpu form list person.pdf

person.pdf
Pg L Field     │ Id | Name        │ Value   │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName   │         │
     Textfield │ 31 | lastName    │         │
     Datefield │ 34 | dob         │         │
     CheckBox  │ 36 | dobVerified │         │
     RadioBGr. │ 17 | gender      │         │ female,male,non-binary
     Textfield │ 38 | country     │         │
   * Textfield │ 40 | planet      │ Earth   │
     Textfield │ 41 | occup       │         │
     ComboBox  │ 42 | status      │ unknown │ alive,deceased,imprisoned,killed,unknown
     ComboBox  │ 43 | license     │         │ CC BY 4.0,CC BY-SA 2.0,Public Domain,unknown
```