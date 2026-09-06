/*
Package pdfcpu is a PDF processing library written in Go supporting encryption.
It provides an API and a command line interface. Supported are all versions up to PDF 2.0 (ISO-32000-2:2020).

The commands are:

	annotations   List, remove page annotations
	attachments   List, add, remove, extract embedded file attachments
	booklet       Arrange pages onto larger sheets of paper to make a booklet or zine
	bookmarks     List, import, export, remove bookmarks
	boxes         List, add, remove page boundaries for selected pages
	certificates  List, inspect, import, reset certificates
	changeopw     Change owner password
	changeupw     Change user password
	collect       Create custom sequence of selected pages
	completion    Generate shell completion script
	config        Initialize, list, inspect, validate, reset configuration
	create        Create PDF content including forms via JSON
	crop          Set cropbox for selected pages
	cut           Custom cut pages horizontally or vertically
	decrypt       Remove password protection
	encrypt       Set password protection
	extract       Extract images, fonts, content, pages or metadata
	fonts         Install, list supported fonts, create cheat sheets
	form          List, remove fields, lock, unlock, reset, export, fill form via JSON or CSV
	grid          Rearrange pages or images for enhanced browsing experience
	help          Help about any command
	images        List, extract, update images
	import        Import/convert images to PDF
	info          Print file info
	keywords      List, add, remove keywords
	merge         Concatenate PDFs
	ndown         Cut selected page into n pages symmetrically
	nup           Rearrange pages or images for reduced number of pages
	optimize      Optimize PDF by getting rid of redundant page resources
	pagelayout    List, set, reset page layout for opened document
	pagemode      List, set, reset page mode for opened document
	pages         Insert, remove selected pages
	paper         Print list of supported paper sizes
	permissions   List, set user access permissions
	portfolio     List, add, remove, extract portfolio entries
	poster        Create poster using paper size
	properties    List, add, remove document properties
	resize        Scale selected pages
	rotate        Rotate selected pages
	selectedpages Print definition of the -pages flag
	signatures    Remove signatures, validate integrity
	split         Split up inFile by span or bookmark
	stamp         Add, remove, update text, image or PDF stamps for selected pages
	trim          Create trimmed version of selected pages
	validate      Validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic PDF 2.0 validation
	version       Print version
	viewerpref    List, set, reset viewer preferences
	watermark     Add, remove, update text, image or PDF watermarks for selected pages
	zoom          Zoom in/out of selected pages
*/
package pdfcpu
