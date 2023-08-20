---
layout: default
---

# Reset form fields

This command resets form fields to their default value.
Either supply a list of form field ids taken from the output of `pdfcpu form list` or skip field ids in order to reset the whole form.

The default value needs to be defined during form creation.
If the form field's default value is undefined the field's current value is deleted
for date fields and text fields and checkboxes will be unchecked. 
For radio button groups, comboboxes and listboxes the current selection is cleared.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu form reset inFile [outFile] [fieldID|fieldName]...
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
| fieldID      | form field id       | no
| fieldName    | form field name     | no

<br>

## Examples

Reset the fields with name `firstName1` and `lastName1`:

```
$ pdfcpu form list english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Jackie                   │
     Textfield │ 31 | lastName1  │ Doeby            │ Doe                      │
     Datefield │ 32 | dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ Yes                      │
     Textfield │ 37 | note1      │                  │ This is a sample text.\n │

$ pdfcpu form reset english.pdf firstName1 lastName1

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Joe                      │
     Textfield │ 31 | lastName1  │ Doeby            │ Doeby                    │
     Datefield │ 32 | dob1       │ 01.01.2000       │ 31.12.1999               │
     RadioBGr. │ 33 | gender1    │ male             │ non-binary               │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ San Francisco,Vienna     │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ Sidney                   │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ Yes                      │
     Textfield │ 37 | note1      │                  │ This is a sample text.\n │
```

<br>

Reset the whole form in engish.pdf:

```
$ pdfcpu form reset english.pdf

english.pdf
Pg L Field     │ Id | Name       │ Default          │ Value                    │ Options
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 1   Textfield │ 30 | firstName1 │ Joe              │ Joe                      │
     Textfield │ 31 | lastName1  │ Doeby            │ Doeby                    │
     Datefield │ 32 | dob1       │ 01.01.2000       │ 01.01.2000               │
     RadioBGr. │ 33 | gender1    │ male             │ male                     │ female,male,non-binary
     ListBox   │ 34 | city11     │ Vienna,São Paulo │ Vienna,São Paulo         │ San Francisco,São Paulo,Vienna
     ComboBox  │ 35 | city12     │ San Francisco    │ San Francisco            │ London,San Francisco,Sidney
     CheckBox  │ 36 | cb11       │                  │ No                       │
     Textfield │ 37 | note1      │                  │                          │
```

