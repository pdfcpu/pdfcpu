/*

Package pdflib is a simple PDF processing library written in Go.
It provides both an API and a command line tool.
Supported are versions up to PDF 1.7 (ISO-32000).

The available commands are:

	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	optimize	optimize PDF by getting rid of redundant page resources
	split		split multi-page PDF into several single-page PDFs
	merge		concatenate 2 or more PDFs
	extract		extract images, fonts, content, pages out of a PDF
	trim		create trimmed version of a PDF
	version		print pdflib version

*/
package pdflib
