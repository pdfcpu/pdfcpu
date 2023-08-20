---
layout: default
---

# Lock form fields

This command makes form fields read-only.
Either supply a list of form field ids taken from the output of `pdfcpu form list` or skip field ids in order to lock the whole form.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu form lock inFile [outFile] [fieldID|fieldName]...
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
| outFile      | PDF output file for dry runs    | no
| fieldID      | form field id       | no
| fieldName    | form field name     | name

<br>

## Examples

Lock the field with name **dob1**:

```
$ pdfcpu form lock english.pdf dob1
writing english.pdf...

$ pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
     Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
   * Datefield │ 32 | dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ Yes                      │
     Textfield │ 37 | note1      │                  │ This is a sample text.\n │
```
<br>

Lock all form fields making the form read-only:

```
$ pdfcpu form lock english.pdf
writing english.pdf...

$ pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1 * Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
   * Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
   * Datefield │ 32 | dob1       │ 01.01.2000       │ 31.12.1999               │
   * RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
   * ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
   * ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
   * CheckBox  │ 36 | cb11       │                  │ Yes                      │
   * Textfield │ 37 | note1      │                  │ This is a sample text.\n │
```