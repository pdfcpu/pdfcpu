---
layout: default
---

# The Create JSON Structure

* Global: paper size, background color etc.  
* Attribute: font, border, margin, padding, color, dir, file
* Primitives: bar, text, box, image, table, form fields

A number of [global definitions](#globals), [attribute](#attributepools) and [primitive pools](#primitivepools) followed by a page dictionary consisting of `page number:page definition` key:value pairs:

```
{
	"paper": "A4",
	...
	"pages": {
		"1": {...},
		"2": {...},
		"3": {...}
	}
}
```

You may either create a new PDF file or append or overlay existing pages.

## Page Sequence Gaps

There may be gaps in the defined page sequence:
```
{
	"paper": "A4",
	...
	"pages": {
		"1": {...},
		"3": {...}
	}
}
```

* If you provide an existing input file, missing page definitions implied by page sequence gaps will only be appended but not modified.<br>

* If you create a new PDF file, missing page definitions will result in blank pages.

## Globals

Global flags & defaults

| name           | description                 | type   | default | required
|:---------------|:----------------------------|--------|---------|----
| origin         | coordinate system           | string | LL      | no
| contentBox     | highlight crop & contentbox | bool   | false   | no
| debug          | highlight positions         | bool   | false   | no
| guides         | render layout guidelines      | bool   | false   | no
| timestamp      | current timestamp format    | string | config  | no
| dateFormat     | current date format         | string | config  | no



Global page defaults:

| name           | description              | type   | default   | required
|:---------------|:-------------------------|--------|-----------|----
| paper          | size                     | string | -         | yes
| crop           | crop box                 | string | media box | no
| bgcol          | background color         | string | -         | no
| border         | border                   | obj    | -         | no
| margin         | margin                   | obj    | -         | no
| padding        | padding                  | obj    | -         | no

You may also define `header` and `footer` as part of the global section.
## AttributePools

Different pages may share attributes like colors, fonts etc.<br>
Attribute pools defined outside of the page sequence contain the definitions of `named attributes` and may serve as templates for inheritance.
Eg. you may overwrite the font size or color at page/content level.<br>
You may reference `named attributes` using `name: $myName`.

| name           | description              
|:---------------|:------------------
| colors         | color pool/templates
| dirs           | dir pool/templates                    
| files          | file pool/templates   
| fonts          | font pool/templates    
| borders        | border pool/templates    
| margins        | margin pool/templates     
| paddings       | padding pool/templates    

## PrimitivePools

Different pages may share primitives like text, images etc.<br>
Primitive pools defined outside of the page sequence contain the definitions of `named primitives` and may serve as  templates for inheritance. Eg. you may overwrite the background color or font for a `text` at page/content level.<br>
You may reference `named primitives` using `name: $myName`.

| name           | description              
|:---------------|:--------------------
| boxes          | box pool/templates        
| images         | imagebox pool/templates      
| texts          | textbox pool/templates   
| tables         | table pool/templates    
| fieldgroups    | form element group pool/templates  

## Page Definition

A number of [page defaults](#globals), [attribute](#attributepools) and [primitive pools](#primitivepools) followed by the [content](#content) element.


## Content

Optional [attribute](#attributepools) and [primitive pools](#primitivepools) followed by arrays for each used primitive containing the positioned primitive instances aka. content elements:

| name   | description              
|:-------|:-----------
| bar    | horizontal/vertical bar
| box    | simple box / rectangular region        
| image  | image box     
| text   | text box  
| table  | table  
 
For PDF form creation you may also add form fields:

| name             | description              
|:-----------------|:--------------------
| textfield        | text input field       
| datefield        | date input field     
| checkbox         | checkbox input field   
| radiobuttongroup | radiobutton input field   
| listbox          | single/multi list selection  
| combobox         | dropdown selection 
| fieldgroup       | container for associated form fields  

### Guides

You may also define an array of guides supporting you during your page design. A guide is a haircross at a specified position within your content box.<br>
You enable your guides using the global `guides` flag.

## Getting Started
We start out by creating a simple page using A6 in landscape mode.
We use the predefined coordinate system with its origin in the lower left corner of the content box and add a single text box positioned at (50/40) using 24 point Helvetica:

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g1shot.png" width="250"><br>
  <img style="border-color:silver" border="1" src="resources/g1.png" width="250">
</p>

<br>

## Page Border

Let's add a page border..<br>
We define the global border element which serves as default border for all pages. The page border is not part of the content box:

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g2shot.png" width="250"><br>
 <img style="border-color:silver" border="1" src="resources/g2.png" width="250">
</p>

<br>

## Page Margin And Padding

Let's add page margin and padding..<br>
We define the global margin and padding elements which serve as default margin and padding for all pages:

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g3shot.png" width="250"><br>
 <img style="border-color:silver" border="1" src="resources/g3.png" width="250">
</p>

<br>

## ContentBox

We highlight the content box in red by turning on the global flag `contentBox`. This flag also highlights the crop box in green.<br>

If you have not specified a crop box using `crop` your crop box defaults to your media box which corresponds to the dimensions of your chosen paper size.<br>
* The page margin separates the page border from the crop box.<br>
* The page padding separates the page border from the content region.<br>
* All three regions are not part of the content region.<br>

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g4shot.png" width="250"><br>
  <img style="border-color:silver" border="1" src="resources/g4.png" width="250">
</p>

<br>

## Debug

We highlight the position of all content elements by turning on the global flag `debug`.<br>
This is especially useful during the layout phase when using different alignments.

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g5shot.png" width="250"><br>
  <img style="border-color:silver" border="1" src="resources/g5.png" width="250">
</p>

<br>

## Origin

As already explained the default coordinate system has its origin in the lower left corner of the content box. You may choose either corner of the content box instead but beware this does not change the fact that an elements position usually corresponds to its lower left corner.<br>
Let's switch the origin to the upper left corner. We can achieve this using `ul` or `upperleft` and are not case sensitive:

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g6shot.png" width="250"><br>
  <img style="border-color:silver" border="1" src="resources/g6.png" width="250">
</p>

<br>

## Guides

There are a couple of pdfcpu features supporting you throughout your design phase. We already discussed `contentbox` and `debug`. An important part during layouting is defining your layout regions.<br>

A haircross is a pair of horizontal and vertical lines intersecting at a certain position, let's call this a `guide`. Using a couple of `guides` helps you keeping track of your layout regions.<br>

`Guides` is an array of elements wrapping `guide` positions.<br>

You need to enable guides rendering by turning on the global flag `guides`.

If you use -1 for one of the position coordinates pdfcpu will apply the center position for content box width/height:


<p align="center">
 <img style="border-color:silver" border="1" src="resources/g7shot.png" width="250"><br>
  <img style="border-color:silver" border="1" src="resources/g7.png" width="250">
</p>

<br>

## Putting all together

Let's finish up by extending this JSON in order to demonstrate text alignment.<br>

`align` is an attribut of `text`.<br>
The possible values are: `left`, `center`, `right`, `justify`<br>
The default alignment is `left`<br><br>
We want to render a short text in the center of the page with three different alignments.<br>
We need three text boxes with corresponding alignment and also want to use different font colors.<br>
The rest of the used `text` attributes `value`, `pos` and `font` are all the same.<br>
Let's use a named text box defining `value`, `pos` and `font` and call it `sample1`.<br><br>
We also want to render a multi line text box using all four possible alignments.<br>
This time we will use individual positions, alignment and width.<br>
The rest of the used `text` attributes `value`, `font`, `bgcol`, `padding` and `border` are shared.<br>
Let's use a named text box defining `value`, `font`, `bgcol`, `padding` and `border` and call it `sample2`.<br><br>
We want to use Helvetica for all text boxes, so we define a named font and call it `myFont`.<br>
Font size and color will be overriden appropriately either within the text pool (`texts`) or within the final content elements:

<p align="center">
 <img style="border-color:silver" border="1" src="resources/g81shot.png" width="350"><br>
 <img style="border-color:silver" border="1" src="resources/g82shot.png" width="350"><br>
 <img style="border-color:silver" border="1" src="resources/g83shot.png" width="350"><br>
  <img style="border-color:silver" border="1" src="resources/g8.png" width="350">
</p>

<br>
