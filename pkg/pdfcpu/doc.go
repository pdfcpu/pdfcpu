/*
Package pdfcpu is a PDF processing library written in Go supporting encryption.
It provides an API and a command line interface. Supported are all versions up to PDF 1.7 (ISO-32000).

The commands are:

	annotations   list, remove page annotations
	attachments   list, add, remove, extract embedded file attachments
	booklet       arrange pages onto larger sheets of paper to make a booklet or zine
	bookmarks     list, import, export, remove bookmarks
	boxes         list, add, remove page boundaries for selected pages
	changeopw     change owner password
	changeupw     change user password
	collect       create custom sequence of selected pages
	config        print configuration
	create        create PDF content including forms via JSON
	crop          set cropbox for selected pages
	cut           custom cut pages horizontally or vertically
	decrypt       remove password protection
	encrypt       set password protection
	extract       extract images, fonts, content, pages or metadata
	fonts         install, list supported fonts, create cheat sheets
	form          list, remove fields, lock, unlock, reset, export, fill form via JSON or CSV
	grid          rearrange pages or images for enhanced browsing experience
	images        list images for selected pages
	import        import/convert images to PDF
	info          print file info
	keywords      list, add, remove keywords
	merge         concatenate PDFs
	ndown         cut selected pages into n pages symmetrically
	nup           rearrange pages or images for reduced number of pages
	optimize      optimize PDF by getting rid of redundant page resources
	pagelayout    list, set, reset page layout for opened document
	pagemode      list, set, reset page mode for opened document
	pages         insert, remove selected pages
	paper         print list of supported paper sizes
	permissions   list, set user access permissions
	portfolio     list, add, remove, extract portfolio entries with optional description
	poster        cut selected pages into poster using paper size or dimensions
	properties    list, add, remove document properties
	resize        scale selected pages
	rotate        rotate selected pages
	selectedpages print definition of the -pages flag
	split         split up a PDF by span or bookmark
	stamp         add, remove, update Unicode text, image or PDF stamps for selected pages
	trim          create trimmed version of selected pages
	validate      validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic PDF 2.0 validation
	version       print version
	viewpref      list, set, reset viewer preferences for opened document
	watermark     add, remove, update Unicode text, image or PDF watermarks for selected pages
*/
package pdfcpu
