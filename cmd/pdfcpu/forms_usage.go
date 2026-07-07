/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

const (
	usageLongForm = `Manage PDF forms.

           inFile ... input PDF file, use - to read from stdin
       inFileData ... input CSV or JSON file
       inFileJSON ... input JSON file
          outFile ... output PDF file, use - to write to stdout where the command writes a single PDF
      outFileJSON ... output JSON file
             json ... output form export JSON
             mode ... output mode (defaults to single)
           outDir ... output directory
          outName ... base output name
          fieldID ... as indicated by "pdfcpu form list"
        fieldName ... as indicated by "pdfcpu form list"

The output modes are:

    single ... each filled form instance gets written to a separate output file.

    merge  ... all filled form instances are merged together resulting in one output file.


Supported usecases:

   1) Get a list of form fields:
         "pdfcpu form list in.pdf" returns a list of form fields of in.pdf.
         Each field is identified by its name and id.
         "pdfcpu form list --json in.pdf" writes the form export JSON to stdout.
         "pdfcpu form list --json *.pdf" writes one fill-compatible form entry per input file.

   2) Remove some form fields:
         "pdfcpu form remove in.pdf middleName birthPlace" removes the the two fields "middleName" and "birthPlace".
         You may supply a mixed list of field ids and field names.

   3) Make some or all fields read-only:
         "pdfcpu form lock in.pdf dateOfBirth" turns the field "dateOfBirth" into read-only.
         "pdfcpu from lock in.pdf" makes the form read-only.
         You may supply a mixed list of field ids and field names.

   4) Make some or all read-only fields writable:
         "pdfcpu form unlock in.pdf dateOfBirth" makes the field "dateOfBirth" writable.
         "pdfcpu form unlock in.pdf" makes all fields of in.pdf writable.
         You may supply a mixed list of field ids and field names.

   5) Clear some or all fields:
         "pdfcpu form reset in.pdf firstName lastName" resets the fields "firstName" and "lastName" to its default values.
         "pdfcpu form reset in.pdf" resets the whole form of in.pdf.
         You may supply a mixed list of field ids and field names.

   6) Export all form fields as preparation for form filling:
         "pdfcpu form export in.pdf" exports field data into a JSON structure written to in.json.

   7) Fill a form with data:
         a) Export your form into in.json and edit the field values.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form fill in.pdf in.json out.pdf" fills in.pdf with form data from in.json and writes the result to out.pdf.

   or

   8) Generate a sequence of filled instances of a form:
         a) Export your form to in.json and edit the field values.
            Extend the JSON Array containing the form by using copy & paste and edit the corresponding form data.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form multifill in.pdf in.json outDir" creates a separate PDF for each filled form instance in outDir.
      or
         a) Export your form to in.json.
         b) Create a CSV file holding form instance data where each CSV line corresponds to one form data tuple.
            The first line identifies fields via id or name from in.json.
         c) "pdfcpu form multifill in.pdf in.csv outDir" creates a separate PDF for each filled form instance in outDir.

   or

   9) Generate a sequence of filled instances of a form and merge output:
         a) Export your form to in.json and edit the field values.
            Extend the JSON Array containing the form by using copy & paste and edit the corresponding form data.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form multifill -m merge in.pdf in.json outDir" creates a single output PDF in outDir.
      or
         a) Export your form to in.json.
         b) Create a CSV file holding form instance data where each CSV line corresponds to one form data tuple.
            The first line identifies fields via id or name in in.json.
         c) "pdfcpu form multifill -m merge in.pdf in.csv outDir" creates a single output PDF in outDir.

Pipeline examples:
   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form list -

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form list --json -

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form export - application.json

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form fill - application.json - \
      | aws s3 cp - s3://acme-forms/application-filled.pdf

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form lock - - \
      | aws s3 cp - s3://acme-forms/application-locked.pdf

   aws s3 cp s3://acme-forms/application-locked.pdf - \
      | pdfcpu form unlock - - \
      | aws s3 cp - s3://acme-forms/application-unlocked.pdf

   aws s3 cp s3://acme-forms/application-filled.pdf - \
      | pdfcpu form reset - - \
      | aws s3 cp - s3://acme-forms/application-reset.pdf

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form remove - - legacyField \
      | aws s3 cp - s3://acme-forms/application-pruned.pdf

   aws s3 cp s3://acme-forms/application.pdf - \
      | pdfcpu form multifill -m merge - applications.csv /tmp/pdfcpu-forms - \
      | aws s3 cp - s3://acme-forms/applications-merged.pdf

   For multifill, outDir is still used for generated form instances even when the merged PDF is written to stdout.

   (For syntax and details please refer to pdfcpu/pkg/api/test/form_test.go)`
)
