---
layout: default
---

# Multifill form via JSON or CSV

This command fills form fields with data via JSON or CSV.

The workflow is similar to [simple form filling](form_fill.md)
except here we import a collection of form instances and generate one PDF for each.

Optionally this command can merge the output PDFs together.

Have a look at some [examples](#examples). 

## Usage

```
pdfcpu form multifill [-m(ode) single|merge] inFile inFileData outDir [outName]
```
<br>

### Flags

| name                             | description               | required   | values
|:---------------------------------|:--------------------------|:-----------|:-
| m(ode)                           | output mode (defaults to single) | no         | single, merge

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

| name         | description                        | required
|:-------------|:-----------------------------------|:--------
| inFile       | PDF input file containing form     | yes
| inFileData   | JSON/CSV input file with form data | yes
| outDir       | output directory                   | yes
| outName      | output file name                   | no

<br>

## Examples

### Multifill via JSON

You can generate your JSON for bulk form fills in different ways.
The workflow steps are:

#### 1. Export your form into JSON using
```
pdfcpu form export
```

#### 2. Remove all fields which shall remain untouched.

#### 3. Copy & paste the form element within the `forms` array.

#### 4. Edit `value` (or `values` where appropriate) for all fields in all form instances.

#### 5. In addition to modifying `value(s)` you may change the `locked` status for fields.

#### 6. To trigger form filling run 
```
pdfcpu form multifill in.pdf in.json outDir
```

#### 7. or if you are only interested in a single output file run
```
pdfcpu form multifill -m merge in.pdf in.json outDir
```

### Multifill via CSV

Here the basic idea is to represent a form instance with a single CSV line in your input data file.
Compared to the JSON way this will reduce the input file size dramatically but it has its limitations when it comes to expressiveness.

The workflow steps are:

#### 1. Export your form into JSON using
```
pdfcpu form export
```

#### 2. Generate a CSV file based on the JSON file you just created and individual form data.
Values prefixed with * will be locked.
Each column represents a form field identified in the header line by field id:

|firstName  |lastName  |dob       |gender     |city         |country
|:----------|:---------|:---------|:----------|-------------|-------
|Jane       |Doe       |06.01.2000|*female    |San Francisco|USA
|Joe        |Miller    |30.07.2001|*male      |São Paulo    |Brazil
|Jackie     |Carson    |29.11.1965|*non-binary|Vienna       |Austria

#### 3. To trigger form filling run
```
pdfcpu form multifill in.pdf in.csv outDir
```

#### 4. or if you are only interested in a single output file run
```
pdfcpu form multifill -m merge in.pdf in.csv outDir
```




