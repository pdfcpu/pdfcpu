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
	passwordFileUsage = `

Password files:
   Use --upw-file or --opw-file wherever the corresponding literal flag is available.
   Supply each password either directly or through a file, but not both.
   Password-file options require a filename; they cannot read from stdin.
   One trailing line ending (LF or CRLF), commonly added by text editors, is allowed and ignored,
   eg. both "abcde" and "abcde" followed by <Enter> supply the password "abcde".
   Spaces and any additional line endings are part of the password.
   An empty file supplies an empty password.
   Encryption and owner-password replacement require a non-empty owner password.`

	usageLongPerm = `Manage user access permissions.

      perm ... user access permissions
    inFile ... input PDF file, use - to read from stdin
   outFile ... output PDF file, use - to write to stdout

   perm modes:
           none: 000000000000 (x000)
          print: 100000000100 (x804)
            all: 111100111100 (xF3C)

   or perm explicitly:
         'x' + max. 3 hex digits (max3Hex, eg. xF30)
         max. 12 binary digits (max12Bits, eg. 111100110000)

   using the permission bits:

      1:  -
      2:  -
      3:  Print (security handlers rev.2), draft print (security handlers >= rev.3)
      4:  Modify contents by operations other than controlled by bits 6, 9, 11.
      5:  Copy, extract text & graphics
      6:  Add or modify annotations, fill form fields, in conjunction with bit 4 create/mod form fields.
      7:  -
      8:  -
      9: Fill form fields (security handlers >= rev.3)
     10: Copy, extract text & graphics (security handlers >= rev.3) (unused since PDF 2.0)
     11: Assemble document (security handlers >= rev.3)
     12: Print (security handlers >= rev.3)

Pipeline examples:
   aws s3 cp s3://acme-legal/protected.pdf - \
      | pdfcpu permissions list -

   aws s3 cp s3://acme-legal/protected.pdf - \
      | pdfcpu permissions set --opw "$OPW" --perm print - - \
      | aws s3 cp - s3://acme-legal/printable.pdf` + passwordFileUsage

	usageLongEncrypt = `Setup password protection based on user and owner password.

      mode ... algorithm (default=aes)
       key ... key length in bits (default=256)
      perm ... user access permissions
    inFile ... input PDF file, use - to read from stdin
   outFile ... output PDF file, use - to write to stdout

PDF 2.0 files have to be encrypted using aes/256.

Pipeline example:
   aws s3 cp s3://acme-hr/onboarding.pdf - \
      | pdfcpu encrypt --opw "$OPW" --upw "$UPW" - - \
      | aws s3 cp - s3://acme-hr/secure/onboarding.pdf` + passwordFileUsage

	usageLongDecrypt = `Remove password protection and reset permissions.

    inFile ... input PDF file, use - to read from stdin
   outFile ... output PDF file, use - to write to stdout

Pipeline example:
   aws s3 cp s3://acme-hr/secure/onboarding.pdf - \
      | pdfcpu decrypt --upw "$UPW" - - \
      | aws s3 cp - s3://acme-hr/plain/onboarding.pdf` + passwordFileUsage

	usageLongChangeUserPW = `Change the user password also known as the open doc password.

       opw ... current owner password (--opw or --opw-file); omission tries an empty password
    inFile ... input PDF file, use - to read from stdin
    upwOld ... old user password
    upwNew ... new user password
   outFile ... output PDF file, use - to write to stdout

File-based password change:
   pdfcpu changeupw in.pdf --opw-file /run/secrets/opw \
      --upwold-file /run/secrets/old-password --upwnew-file /run/secrets/new-password out.pdf

   Supply both --upwold-file and --upwnew-file together, with inFile [outFile].
   Omit positional old and new passwords when using these file options.
   Both current passwords are authenticated before changing either password.

Pipeline example:
   aws s3 cp s3://acme-legal/client.pdf - \
      | pdfcpu changeupw - --opw "$OPW" "$OLD_UPW" "$NEW_UPW" - \
      | aws s3 cp - s3://acme-legal/client-rotated-upw.pdf` + passwordFileUsage

	usageLongChangeOwnerPW = `Change the owner password also known as the set permissions password.

       upw ... current user password (--upw or --upw-file); omission tries an empty password
    inFile ... input PDF file, use - to read from stdin
    opwOld ... current owner password
    opwNew ... new owner password
   outFile ... output PDF file, use - to write to stdout

File-based password change:
   pdfcpu changeopw in.pdf --upw-file /run/secrets/upw \
      --opwold-file /run/secrets/old-password --opwnew-file /run/secrets/new-password out.pdf

   Supply both --opwold-file and --opwnew-file together, with inFile [outFile].
   Omit positional old and new passwords when using these file options.
   Both current passwords are authenticated before changing either password.

Pipeline example:
   aws s3 cp s3://acme-legal/client.pdf - \
      | pdfcpu changeopw - --upw "$UPW" "$OLD_OPW" "$NEW_OPW" - \
      | aws s3 cp - s3://acme-legal/client-rotated-opw.pdf` + passwordFileUsage
)
