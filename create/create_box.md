---
layout: default
---

# Render Boxes

# box
This primitive renders a rectangular region.

A box has a position.
The position may be defined by the coordinates of its lower left corner:
"pos": [100,100]

A box max also be anchored and thereby positioned::
"anchor": 

You may nudge the position by some horizontal/vertical delta dx/dy
"dx": 50,
"dy": 50,

A box needs a width and a height.

A box may have a border:
A Border has a width, a color and an optional style:
"border": { "name": "$myBorder",
            "width: 100,
			"col": "White" or "#1E90FF"
            "style: miter\", \"round\" or \"bevel
		  },

A box may have a margin:
"margin": { "width": 10 },

You may specify a fill color.
"fillCol": "black"

You may specify a rotation angle:
"rot":  90

Finally a named box may serve as a template.
"name": "$colorBox"

You disable rendering the box like with all primitives: "hide":true





<p align="center">
  <img style="border-color:silver" border="1" src="resources/boxes.png" width="90%">
</p>

```
{
	"paper": "A4L",
	"colors": {
		"DodgerBlue": "#1E90FF",
		"Beige": "#F5F5DC",
	},
	"margin": {
		"width": 10
	},
	"borders": {
		"border": {
			"width": 10,
			"col": "$DodgerBlue"
		},
		"myBorder": {
			"width": 5,
			"col": "#FF0000",
			"style": "round"
		}
	},
	"padding": {
		"width": 10
	},
	"pages": {
		"1": {
			"bgcol": "#8fbc2f",
			"content": {
				"bgcol": "#8fbcff",
				"border": {
					"width": 10,
					"col": "#698b69",
					"style": "round"
				},
				"box": [
					{
						"hide": false,
						"comment": "Empty box",
						"anchor": "topLeft",
						"width": 200,
						"height": 100,
						"border": {
							"col": "black"
						}
					},
					{
						"comment": "Filled box with border",
						"anchor": "topCenter",
						"width": 200,
						"height": 100,
						"fillCol": "$Beige",
						"border": {
							"width": 10,
							"col": "#032890"
						}
					},
					{
						"comment": "Filled box without border",
						"anchor": "topRight",
						"width": 200,
						"height": 100,
						"fillCol": "$Beige"
					},
					{
						"comment": "Filled box without border",
						"anchor": "left",
						"width": 200,
						"height": 100,
						"fillCol": "$beige",
						"border": {
							"col": "black"
						}
					},
					{
						"comment": "Filled box with rounded border",
						"anchor": "center",
						"width": 200,
						"height": 100,
						"fillCol": "#AA2890",
						"border": {
							"width": 10,
							"col": "#0022AA",
							"style": "round"
						}
					},
					{
						"comment": "Empty box with border",
						"anchor": "right",
						"width": 200,
						"height": 100,
						"border": {
							"name": "$myBorder"
						}
					},
					{
						"comment": "Empty box with border",
						"anchor": "bottomLeft",
						"width": 200,
						"height": 100,
						"border": {
							"width": 10,
							"col": "#00AA00",
							"style": "round"
						}
					},
					{
						"comment": "Empty box, red border",
						"anchor": "bottomCenter",
						"width": 200,
						"height": 100,
						"border": {
							"width": 0,
							"col": "#FF0000"
						}
					},
					{
						"comment": "Filled box with border",
						"anchor": "bottomRight",
						"width": 200,
						"height": 100,
						"fillCol": "#032890",
						"border": {
							"width": 10,
							"col": "#00AA00"
						}
					}
				]
			}
		}
	}
}
```

