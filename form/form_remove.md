---
layout: default
---

# Remove form fields

This command removes form fields by their id
taken from the output of `pdfcpu form list`.

Although the optional field label is an attribute of the 
JSON form field element, this command removes the field only.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu form remove inFile [outFile] <fieldID|fieldName>...
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
| outFile      | PDF output file for dry runs     | no
| fieldID      | form field id       | either
| fieldName    | form field name     | or

<br>

## Examples

Remove the field with id **32**:

```
pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
     Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
   * Datefield │ 32 | dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ Yes                      │
     Textfield │ 37 | note1      │                  │ This is a sample text.\n │

pdfcpu form remove english.pdf 32
writing english.pdf...

pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
     Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
     RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ Yes                      │
     Textfield │ 37 | note1      │                  │ This is a sample text.\n │
```
