/*
Copyright 2019 The pdf Authors.

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

package test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var sampleText string = `MOST of the adventures recorded in this book really occurred; one or
two were experiences of my own, the rest those of boys who were
schoolmates of mine. Huck Finn is drawn from life; Tom Sawyer also, but
not from an individual--he is a combination of the characteristics of
three boys whom I knew, and therefore belongs to the composite order of
architecture.

The odd superstitions touched upon were all prevalent among children
and slaves in the West at the period of this story--that is to say,
thirty or forty years ago.

Although my book is intended mainly for the entertainment of boys and
girls, I hope it will not be shunned by men and women on that account,
for part of my plan has been to try to pleasantly remind adults of what
they once were themselves, and of how they felt and thought and talked,
and what queer enterprises they sometimes engaged in.`

var sampleTextArabic = `حدثت بالفعل معظم المغامرات المسجلة في هذا الكتاب ؛ واحد أو
كانت اثنتان من تجربتي الخاصة ، والباقي تجارب الأولاد الذين كانوا كذلك
زملائي في المدرسة. هاك فين مستوحى من الحياة ؛ توم سوير أيضا ولكن
ليس من فرد - إنه مزيج من خصائص
ثلاثة أولاد أعرفهم ، وبالتالي ينتمون إلى الترتيب المركب لـ
هندسة معمارية.

كانت الخرافات الغريبة التي تم التطرق إليها سائدة بين الأطفال
والعبيد في الغرب في فترة هذه القصة - أي
قبل ثلاثين أو أربعين سنة.

على الرغم من أن كتابي مخصص بشكل أساسي للترفيه عن الأولاد و
الفتيات ، أتمنى ألا يتجنب الرجال والنساء ذلك الحساب ،
جزء من خطتي كان محاولة تذكير البالغين بما يحدث
كانوا أنفسهم ذات مرة ، وكيف شعروا وفكروا وتحدثوا ،
وما هي المؤسسات الكويرية التي شاركوا فيها أحيانًا.`

var sampleTextHebrew = `רוב ההרפתקאות שתועדו בספר זה באמת התרחשו; אחד או
שתיים היו חוויות משלי, והשאר אלה של בנים שהיו
חברי לבית הספר שלי. האק פין נשאב מהחיים; גם טום סוייר, אבל
לא מאדם - הוא שילוב של המאפיינים של
שלושה בנים שהכרתי ולכן שייכים לסדר המורכב של
ארכיטקטורה.

האמונות הטפלות המוזרות בהן נגעו היו כולן רווחות בקרב ילדים
ועבדים במערב בתקופת הסיפור הזה - כלומר,
לפני שלושים או ארבעים שנה.

למרות שהספר שלי מיועד בעיקר לבידור של בנים ו
בנות, אני מקווה שזה לא יימנע מגברים ונשים בגלל זה,
חלק מהתוכנית שלי הייתה לנסות להזכיר למבוגרים בנעימות מה
פעם הם היו עצמם, ועל איך שהם הרגישו וחשבו ודיברו,
ובאילו מפעלים מוזרים הם עסקו לפעמים.`

var sampleTextPersian = `بیشتر ماجراهای ثبت شده در این کتاب واقعاً اتفاق افتاده است. یکی یا
دو مورد از تجربه های خودم بود ، بقیه از پسران بودند
هم مدرسه ای های من. هاک فین از زندگی کشیده شده است. تام سویر نیز ، اما
نه از یک فرد - او ترکیبی از ویژگی های است
سه پسر که من آنها را می شناختم و بنابراین به ترتیب مرکب تعلق دارند
معماری.

خرافات عجیب و غریب لمس شده همه در میان کودکان شایع بود
و بردگان در غرب این دوره از داستان - یعنی اینکه ،
سی چهل سال پیش

اگرچه کتاب من عمدتا برای سرگرمی پسران و
دختران ، امیدوارم با این حساب مردان و زنان از آن اجتناب نکنند ،
زیرا بخشی از برنامه من این بوده است که سعی کنم چه چیزی را به بزرگسالان یادآوری کنم
آنها یک بار خودشان بودند ، و از احساس و تفکر و صحبت کردن ،
و بعضی اوقات چه کارهایی را انجام می دادند`

var sampleTextUrdu = `اس کتاب میں درج کی گئی زیادہ تر مہم جوئی واقعتا؛ پیش آئی ہے۔ ایک یا
دو میرے اپنے تجربات تھے ، باقی جو لڑکے تھے
میرے اسکول کے ساتھیوں. ہک فن زندگی سے نکالا گیا ہے۔ ٹام ساویر بھی ، لیکن
کسی فرد سے نہیں - وہ کی خصوصیات کا ایک مجموعہ ہے
تین لڑکے جن کو میں جانتا تھا ، اور اس وجہ سے یہ جامع ترتیب سے ہے
فن تعمیر

بچوں میں عجیب و غریب اندوشواس کا اثر تمام تھا
اور اس کہانی کے دور میں مغرب میں غلام۔
تیس یا چالیس سال پہلے کی بات ہے۔

اگرچہ میری کتاب بنیادی طور پر لڑکوں اور تفریح ​​کے لئے ہے
لڑکیاں ، مجھے امید ہے کہ اس وجہ سے مرد اور خواتین اس سے باز نہیں آئیں گے ،
میرے منصوبے کا ایک حص adultsہ یہ رہا ہے کہ بالغوں کو خوشی سے اس کی یاد دلانے کی کیا کوشش کی جائے
وہ ایک بار خود تھے ، اور یہ کہ وہ کیسے محسوس کرتے ہیں ، سوچتے اور بات کرتے ہیں ،
اور کن کن کن کاروباری اداروں میں وہ کبھی کبھی مشغول رہتے ہیں۔`

var sampleTextRTL = map[string]string{
	"Arabic":  sampleTextArabic,
	"Hebrew":  sampleTextHebrew,
	"Persian": sampleTextPersian,
	"Urdu":    sampleTextUrdu,
}

var sampleText2 = `THE two boys flew on and on, toward the village, speechless with
horror. They glanced backward over their shoulders from time to time,
apprehensively, as if they feared they might be followed. Every stump
that started up in their path seemed a man and an enemy, and made them
catch their breath; and as they sped by some outlying cottages that lay
near the village, the barking of the aroused watch-dogs seemed to give
wings to their feet.

"If we can only get to the old tannery before we break down!"
whispered Tom, in short catches between breaths. "I can't stand it much
longer."

Huckleberry's hard pantings were his only reply, and the boys fixed
their eyes on the goal of their hopes and bent to their work to win it.
They gained steadily on it, and at last, breast to breast, they burst
through the open door and fell grateful and exhausted in the sheltering
shadows beyond. By and by their pulses slowed down, and Tom whispered:

"Huckleberry, what do you reckon'll come of this?"`

var sampleText3 = `Even the Glorious Fourth was in some sense a failure, for it rained
hard, there was no procession in consequence, and the greatest man in
the world (as Tom supposed), Mr. Benton, an actual United States
Senator, proved an overwhelming disappointment--for he was not
twenty-five feet high, nor even anywhere in the neighborhood of it.

A circus came. The boys played circus for three days afterward in
tents made of rag carpeting--admission, three pins for boys, two for
girls--and then circusing was abandoned.

A phrenologist and a mesmerizer came--and went again and left the
village duller and drearier than ever.

There were some boys-and-girls' parties, but they were so few and so
delightful that they only made the aching voids between ache the harder.

Becky Thatcher was gone to her Constantinople home to stay with her
parents during vacation--so there was no bright side to life anywhere.`

func createAndValidate(t *testing.T, xRefTable *model.XRefTable, outFile, msg string) {
	t.Helper()
	outDir := "../../samples/basic"
	outFile = filepath.Join(outDir, outFile)
	if err := api.CreatePDFFile(xRefTable, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCreateDemoPDF(t *testing.T) {
	msg := "TestCreateDemoPDF"
	mediaBox := types.RectForFormat("A4")
	p := model.Page{MediaBox: mediaBox, Fm: model.FontMap{}, Buf: new(bytes.Buffer)}
	pdfcpu.CreateTestPageContent(p)
	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "Test.pdf", msg)
}

func TestResourceDictInheritanceDemoPDF(t *testing.T) {
	// Create a test page proofing resource inheritance.
	// Resources may be inherited from ANY parent node.
	// Case in point: fonts
	msg := "TestResourceDictInheritanceDemoPDF"
	xRefTable, err := pdfcpu.CreateResourceDictInheritanceDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "ResourceDictInheritanceDemo.pdf", msg)
}

func TestAnnotationDemoPDF(t *testing.T) {
	msg := "TestAnnotationDemoPDF"
	xRefTable, err := pdfcpu.CreateAnnotationDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "AnnotationDemo.pdf", msg)
}

func writeTextDemoAlignedWidthAndMargin(
	xRefTable *model.XRefTable,
	p model.Page,
	region *types.Rectangle,
	hAlign types.HAlignment,
	w, mLeft, mRight, mTop, mBot float64) {

	buf := p.Buf
	mediaBox := p.MediaBox

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       24,
		ShowMargins:    true,
		MLeft:          mLeft,
		MRight:         mRight,
		MTop:           mTop,
		MBot:           mBot,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         hAlign,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.Black,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     true,
		ShowTextBB:     true,
		HairCross:      true,
	}

	td.VAlign, td.X, td.Y, td.Text = types.AlignBaseline, -1, r.Height()*.75, "M\\u(lti\nline\n\nwith empty line"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignBaseline, r.Width()*.75, r.Height()*.25, "Arbitrary\ntext\nlines"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	// Multilines along the top of the page:
	td.VAlign, td.X, td.Y, td.Text = types.AlignTop, 0, r.Height(), "0,h (topleft)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignTop, -1, r.Height(), "-1,h (topcenter)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignTop, r.Width(), r.Height(), "w,h (topright)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	// Multilines along the center of the page:
	// x = 0 centers the position of multilines horizontally
	// y = 0 centers the position of multilines vertically and enforces alignMiddle
	td.VAlign, td.X, td.Y, td.Text = types.AlignBaseline, 0, -1, "0,-1 (left)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignMiddle, -1, -1, "-1,-1 (center)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignBaseline, r.Width(), -1, "w,-1 (right)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	// Multilines along the bottom of the page:
	td.VAlign, td.X, td.Y, td.Text = types.AlignBottom, 0, 0, "0,0 (botleft)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignBottom, -1, 0, "-1,0 (botcenter)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = types.AlignBottom, r.Width(), 0, "w,0 (botright)\nand line2"
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, w)

	draw.DrawHairCross(buf, 0, 0, r)
}

func createTextDemoAlignedWidthAndMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle, hAlign types.HAlignment, w, mLeft, mRight, mTop, mBot float64) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextDemoAlignedWidthAndMargin(xRefTable, p, region, hAlign, w, mLeft, mRight, mTop, mBot)
	region = types.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAlignedWidthAndMargin(xRefTable, p, region, hAlign, w, mLeft, mRight, mTop, mBot)
	return p
}

func createTextDemoAlignLeft(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignLeft, 0, 0, 0, 0, 0)
}

func createTextDemoAlignLeftMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignLeft, 0, 5, 10, 15, 20)
}

func createTextDemoAlignRight(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignRight, 0, 0, 0, 0, 0)
}

func createTextDemoAlignRightMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignRight, 0, 5, 10, 15, 20)
}

func createTextDemoAlignCenter(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignCenter, 0, 0, 0, 0, 0)
}

func createTextDemoAlignCenterMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignCenter, 0, 5, 10, 15, 20)
}

func createTextDemoAlignJustify(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignJustify, 0, 0, 0, 0, 0)
}

func createTextDemoAlignJustifyMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignJustify, 0, 5, 10, 15, 20)
}

func createTextDemoAlignLeftWidth(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignLeft, 250, 0, 0, 0, 0)
}

func createTextDemoAlignLeftWidthAndMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignLeft, 250, 5, 10, 15, 20)
}

func createTextDemoAlignRightWidth(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignRight, 250, 0, 0, 0, 0)
}

func createTextDemoAlignRightWidthAndMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignRight, 250, 5, 10, 15, 20)
}

func createTextDemoAlignCenterWidth(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignCenter, 250, 0, 0, 0, 0)
}

func createTextDemoAlignCenterWidthAndMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignCenter, 250, 5, 40, 15, 20)
}

func createTextDemoAlignJustifyWidth(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignJustify, 250, 0, 0, 0, 0)
}

func createTextDemoAlignJustifyWidthAndMargin(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	return createTextDemoAlignedWidthAndMargin(xRefTable, mediaBox, types.AlignJustify, 250, 5, 10, 15, 20)
}

func writeTextAlignJustifyDemo(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, fontName string) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		X:              -1,
		Y:              -1,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         types.AlignJustify,
		VAlign:         types.AlignMiddle,
		RMode:          draw.RMFill,
		StrokeCol:      color.NewSimpleColor(0x206A29),
		FillCol:        color.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)

	draw.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func writeTextAlignJustifyColumnDemo(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Times-Roman"
	fontName2 := "Helvetica"
	k1 := p.Fm.EnsureKey(fontName)
	k2 := p.Fm.EnsureKey(fontName2)

	td := model.TextDescriptor{
		Text:           sampleText,
		Embed:          true,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         types.AlignJustify,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.Black,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.BackgroundCol = color.White
	td.FillCol = color.Black
	td.FontName, td.FontKey, td.FontSize = fontName, k1, 9
	td.ParIndent = true
	td.VAlign, td.X, td.Y, td.Dx, td.Dy = types.AlignTop, 0, r.Height(), 5, -5
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, 150)

	td.BackgroundCol = color.Black
	td.FillCol = color.White
	td.FontName, td.FontKey, td.FontSize = fontName2, k2, 12
	td.ParIndent = true
	td.VAlign, td.X, td.Y, td.Dx, td.Dy = types.AlignTop, -1, -1, 0, 0
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, 290)

	draw.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTextAlignJustifyDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	fontName := "Times-Roman"
	writeTextAlignJustifyDemo(xRefTable, p, region, fontName)
	region = types.RectForWidthAndHeight(0, 0, 200, 200)
	writeTextAlignJustifyDemo(xRefTable, p, region, fontName)
	return p
}

func createTextAlignJustifyColumnDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextAlignJustifyColumnDemo(xRefTable, p, region)
	region = types.RectForWidthAndHeight(0, 0, 200, 200)
	writeTextAlignJustifyColumnDemo(xRefTable, p, region)
	return p
}

func writeTextDemoAnchorsWithOffset(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, dx, dy float64) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       24,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.Black,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     true,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy, td.Text = dx, -dy, "topleft\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft)

	td.Dx, td.Dy, td.Text = 0, -dy, "topcenter\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter)

	td.Dx, td.Dy, td.Text = -dx, -dy, "topright\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight)

	td.Dx, td.Dy, td.Text = dx, 0, "left\nandline2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Left)

	td.Dx, td.Dy, td.Text = 0, 0, "center\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Center)

	td.Dx, td.Dy, td.Text = -dx, 0, "right\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Right)

	td.Dx, td.Dy, td.Text = dx, dy, "botleft\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft)

	td.Dx, td.Dy, td.Text = 0, dy, "botcenter\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter)

	td.Dx, td.Dy, td.Text = -dx, dy, "botright\nandLine2"
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight)

	draw.DrawHairCross(buf, 0, 0, r)
}

func writeTextDemoAnchors(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	writeTextDemoAnchorsWithOffset(xRefTable, p, region, 0, 0)
}

func createTextDemoAnchors(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextDemoAnchors(xRefTable, p, region)
	region = types.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAnchors(xRefTable, p, region)
	return p
}

func createTextDemoAnchorsWithOffset(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	dx, dy := 20., 20.
	var region *types.Rectangle
	writeTextDemoAnchorsWithOffset(xRefTable, p, region, dx, dy)
	region = types.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAnchorsWithOffset(xRefTable, p, region, dx, dy)
	return p
}

func writeTextDemoColumnAnchoredWithOffset(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, dx, dy float64) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	wSmall := 100.
	wBig := 300.

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       6,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.Black,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy = dx, -dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft, wBig)

	td.Dx, td.Dy = 0, -dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter, wBig)

	td.Dx, td.Dy = -dx, -dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight, wBig)

	td.Dx, td.Dy = dx, 0
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Left, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Left, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Left, wBig)

	td.Dx, td.Dy = 0, 0
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Center, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Center, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Center, wBig)

	td.Dx, td.Dy = -dx, 0
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Right, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Right, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.Right, wBig)

	td.Dx, td.Dy = dx, dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft, wBig)

	td.Dx, td.Dy = 0, dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter, wBig)

	td.Dx, td.Dy = -dx, dy
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight, wSmall)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight, 0)
	model.WriteColumnAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight, wBig)

	draw.DrawHairCross(buf, 0, 0, mediaBox)
}

func writeTextDemoColumnAnchored(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	writeTextDemoColumnAnchoredWithOffset(xRefTable, p, region, 0, 0)
}

func createTextDemoColumnAnchored(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextDemoColumnAnchored(xRefTable, p, region)
	region = types.RectForWidthAndHeight(50, 70, 400, 400)
	writeTextDemoColumnAnchored(xRefTable, p, region)
	return p
}

func createTextDemoColumnAnchoredWithOffset(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	dx, dy := 20., 20.
	writeTextDemoColumnAnchoredWithOffset(xRefTable, p, region, dx, dy)
	region = types.RectForWidthAndHeight(50, 70, 400, 400)
	writeTextDemoColumnAnchoredWithOffset(xRefTable, p, region, dx, dy)
	return p
}

func writeTextRotateDemoWithOffset(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, dx, dy float64) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
		draw.DrawHairCross(buf, 0, 0, r)
	}

	fillCol := color.Black

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           "Hello Gopher!\nLine 2",
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       24,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy = dx, -dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)
	td.Rotation, td.FillCol = 45, color.SimpleColor{R: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)
	td.Rotation, td.FillCol = 90, color.SimpleColor{R: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)
	td.Rotation, td.FillCol = 45, color.SimpleColor{G: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)
	td.Rotation, td.FillCol = 90, color.SimpleColor{G: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)
	td.Rotation, td.FillCol = 45, color.SimpleColor{B: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)
	td.Rotation, td.FillCol = 90, color.SimpleColor{B: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)
	td.Rotation, td.FillCol = 45, color.SimpleColor{R: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)
	td.Rotation, td.FillCol = 90, color.SimpleColor{R: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)

	td.Dx, td.Dy = 0, 0
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)
	td.Rotation, td.FillCol = 45, color.SimpleColor{G: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)
	td.Rotation, td.FillCol = 90, color.SimpleColor{G: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)

	td.Dx, td.Dy = -dx, 0
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)
	td.Rotation, td.FillCol = 45, color.SimpleColor{B: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)
	td.Rotation, td.FillCol = 90, color.SimpleColor{B: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)

	td.Dx, td.Dy = dx, dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)
	td.Rotation, td.FillCol = 45, color.SimpleColor{R: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)
	td.Rotation, td.FillCol = 90, color.SimpleColor{R: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)
	td.Rotation, td.FillCol = 45, color.SimpleColor{G: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)
	td.Rotation, td.FillCol = 90, color.SimpleColor{G: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)

	td.Dx, td.Dy = -dx, dy
	td.Rotation, td.FillCol = 0, fillCol
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)
	td.Rotation, td.FillCol = 45, color.SimpleColor{B: 1}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)
	td.Rotation, td.FillCol = 90, color.SimpleColor{B: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)
}

func writeTextRotateDemo(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	writeTextRotateDemoWithOffset(xRefTable, p, region, 0, 0)
}

func createTextRotateDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextRotateDemo(xRefTable, p, region)
	region = types.RectForWidthAndHeight(150, 150, 300, 300)
	writeTextRotateDemo(xRefTable, p, region)
	return p
}

func createTextRotateDemoWithOffset(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	dx, dy := 20., 20.
	writeTextRotateDemoWithOffset(xRefTable, p, region, dx, dy)
	region = types.RectForWidthAndHeight(150, 150, 300, 300)
	writeTextRotateDemoWithOffset(xRefTable, p, region, dx, dy)
	return p
}

func writeTextScaleAbsoluteDemoWithOffset(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, dx, dy float64) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fillCol := color.Black
	bgCol := color.SimpleColor{R: 1., G: .98, B: .77}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.HAlign, td.VAlign, td.X, td.Y, td.FontSize = types.AlignJustify, types.AlignMiddle, -1, r.Height()*.72, 9
	td.Scale, td.FillCol = 1, fillCol
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)
	td.Scale, td.FillCol = 1.5, color.SimpleColor{R: 1}
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)
	td.Scale, td.FillCol = 2, color.SimpleColor{R: .5}
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)

	width := 130.

	td.HAlign, td.VAlign, td.X = types.AlignJustify, types.AlignMiddle, r.Width()*.75
	td.FillCol, td.Text = fillCol, "Justified column\nWidth=130"

	td.FontSize, td.Y = 24, r.Height()*.35
	td.Scale = 1
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)
	td.Scale = 1.5
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)

	td.FontSize, td.Y = 12, r.Height()*.22
	td.Scale = 1
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)
	td.Scale = 1.5
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)

	td.FontSize = 9
	td.Scale, td.Y = 1, r.Height()*.15
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)
	td.Scale, td.Y = 1.5, r.Height()*.13
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)

	td = model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	text15 := "Hello Gopher!\nAbsolute Width=1.5"
	text1 := "Hello Gopher!\nAbsolute Width=1"
	text5 := "Hello Gopher!\nAbsolute Width=.5"

	td.Dx, td.Dy = dx, -dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{R: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{R: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{G: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{G: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{B: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{B: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{R: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)
	td.Scale, td.FillCol = .5, color.SimpleColor{R: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Left)

	td.Dx, td.Dy = 0, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{G: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{G: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Center)

	td.Dx, td.Dy = -dx, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{B: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{B: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.Right)

	td.Dx, td.Dy = dx, dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{R: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)
	td.Scale, td.FillCol = .5, color.SimpleColor{R: .5}
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{G: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{G: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomCenter)

	td.Dx, td.Dy = -dx, +dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)
	td.Scale, td.FillCol, td.Text = 1, color.SimpleColor{B: 1}, text1
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)
	td.Scale, td.FillCol, td.Text = .5, color.SimpleColor{B: .5}, text5
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, r, td, types.BottomRight)

	draw.DrawHairCross(buf, 0, 0, r)
}

func writeTextScaleAbsoluteDemo(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	writeTextScaleAbsoluteDemoWithOffset(xRefTable, p, region, 0, 0)
}

func createTextScaleAbsoluteDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextScaleAbsoluteDemo(xRefTable, p, region)
	region = types.RectForWidthAndHeight(20, 70, 180, 180)
	writeTextScaleAbsoluteDemo(xRefTable, p, region)
	return p
}

func createTextScaleAbsoluteDemoWithOffset(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	dx, dy := 20., 20.
	var region *types.Rectangle
	writeTextScaleAbsoluteDemoWithOffset(xRefTable, p, region, dx, dy)
	region = types.RectForWidthAndHeight(20, 70, 180, 180)
	writeTextScaleAbsoluteDemoWithOffset(xRefTable, p, region, dx, dy)
	return p
}

func writeTextScaleRelativeDemoWithOffset(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, dx, dy float64) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fillCol := color.Black
	bgCol := color.SimpleColor{R: 1., G: .98, B: .77}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         types.AlignJustify,
		VAlign:         types.AlignMiddle,
		X:              -1,
		Y:              r.Height() * .73,
		ScaleAbs:       false,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.FontSize, td.Scale, td.FillCol = 9, .4, fillCol
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)
	td.FontSize, td.Scale, td.FillCol = 9, .6, color.SimpleColor{R: 1}
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)
	td.FontSize, td.Scale, td.FillCol = 9, .8, color.SimpleColor{R: .5}
	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)

	width := 130.

	td = model.TextDescriptor{
		Text:           "Justified column\nWidth=130",
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         types.AlignJustify,
		VAlign:         types.AlignMiddle,
		X:              r.Width() * .75,
		Y:              r.Height() * .25,
		ScaleAbs:       false,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}
	td.Scale, td.FillCol = .5, fillCol
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)
	td.Scale, td.FillCol = .3, color.SimpleColor{G: 1}
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)
	td.Scale, td.FillCol = .20, color.SimpleColor{G: .5}
	model.WriteColumn(xRefTable, buf, mediaBox, region, td, width)

	td = model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         types.AlignJustify,
		VAlign:         types.AlignMiddle,
		X:              r.Width() * .75,
		Y:              r.Height() * .25,
		ScaleAbs:       false,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	text10 := "Hello Gopher!\nRelative Width=10%"
	text20 := "Hello Gopher!\nRelative Width=20%"
	text30 := "Hello Gopher!\nRelative Width=30%"

	td.Dx, td.Dy = dx, -dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{R: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{R: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{G: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{G: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{B: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{B: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Left)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{R: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Left)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{R: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Left)

	td.Dx, td.Dy = 0, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Center)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{G: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Center)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{G: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Center)

	td.Dx, td.Dy = -dx, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Right)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{B: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Right)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{B: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.Right)

	td.Dx, td.Dy = dx, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{R: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{R: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{G: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{G: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomCenter)

	td.Dx, td.Dy = -dx, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight)
	td.Scale, td.FillCol, td.Text = .2, color.SimpleColor{B: 1}, text20
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight)
	td.Scale, td.FillCol, td.Text = .1, color.SimpleColor{B: .5}, text10
	model.WriteMultiLineAnchored(xRefTable, buf, mediaBox, region, td, types.BottomRight)

	draw.DrawHairCross(buf, 0, 0, r)
}

func writeTextScaleRelativeDemo(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) {
	writeTextScaleRelativeDemoWithOffset(xRefTable, p, region, 0, 0)
}

func createTextScaleRelativeDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writeTextScaleRelativeDemo(xRefTable, p, region)
	region = types.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextScaleRelativeDemo(xRefTable, p, region)
	return p
}

func createTextScaleRelativeDemoWithOffset(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	dx, dy := 20., 20.
	writeTextScaleRelativeDemoWithOffset(xRefTable, p, region, dx, dy)
	region = types.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextScaleRelativeDemoWithOffset(xRefTable, p, region, dx, dy)
	return p
}

func createTextDemoColumns(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       9,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		ShowBackground: true,
		BorderWidth:    3,
	}

	// 1st row: 3 side by side columns using anchors, width and a background color.

	width := mediaBox.Width() / 3
	td.MinHeight = mediaBox.Height() / 2

	// Render left column.
	// Draw the bounding box with rounded corners but no borders.
	td.Text = sampleText
	td.ShowTextBB, td.ShowBorder = true, false
	td.BackgroundCol = color.SimpleColor{R: .4, G: .98, B: .77}
	td.BorderStyle = types.LJRound
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, nil, td, types.TopLeft, width)

	// Render middle column.
	// Draw the bounding box with regular corners but no border.
	td.Text = sampleText2
	td.Dx = -width / 2
	td.ShowTextBB, td.ShowBorder = true, false
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJMiter
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, nil, td, types.TopCenter, width)

	// Render right column.
	// Draw bounding box and a border with rounded corners.

	td.Text = sampleText3
	td.Dx = 0
	td.ShowTextBB, td.ShowBorder = true, true
	td.BackgroundCol = color.SimpleColor{R: 1., G: .98, B: .77}
	td.BorderCol = color.SimpleColor{R: .2, G: .5, B: .2}
	td.BorderStyle = types.LJRound
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, nil, td, types.TopRight, width)

	// 2nd row: 3 side by side columns below using relative scaling,
	// Indent paragraph beginnings and don't draw the background.
	relScaleFactor := .334
	td.Dy = mediaBox.Height() / 2
	td.Scale = relScaleFactor
	td.ScaleAbs = false
	td.ParIndent = true
	td.ShowBackground, td.ShowBorder = false, true
	td.HAlign, td.VAlign = types.AlignJustify, types.AlignTop

	// Render left column.
	td.Text = sampleText
	td.X = 0
	td.ShowTextBB = true
	td.BorderStyle = types.LJBevel
	model.WriteMultiLine(xRefTable, p.Buf, mediaBox, nil, td)

	// Render middle column.
	td.Text = sampleText2
	td.X = mediaBox.Width() / 2
	td.Dx = -width / 2
	td.ShowTextBB = false
	model.WriteMultiLine(xRefTable, p.Buf, mediaBox, nil, td)

	// Render right column.
	td.Text = sampleText3
	td.X = mediaBox.Width()
	td.Dx = 0
	td.ShowTextBB = true
	td.BorderStyle = types.LJMiter
	model.WriteMultiLine(xRefTable, p.Buf, mediaBox, nil, td)

	draw.DrawHairCross(p.Buf, 0, 0, mediaBox)

	return p
}

func writeTextBorderTest(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle) model.Page {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		FontName:   fontName,
		Embed:      true,
		FontKey:    k,
		FontSize:   7,
		MLeft:      10,
		MRight:     10,
		MTop:       10,
		MBot:       10,
		Scale:      1.,
		ScaleAbs:   true,
		RMode:      draw.RMFill,
		BorderCol:  color.NewSimpleColor(0xabe003),
		ShowTextBB: true,
	}

	w := mediaBox.Width() / 2

	// no background, no margin, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = false, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BorderWidth = 0
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJMiter
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.TopLeft, w)

	// with background, no margin, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BorderWidth = 0
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.TopCenter, w)

	// with background, with margins, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.Dy = 100
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.Left, w)

	// with background, with margins, show margins, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, true
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJMiter
	td.Dy = 100
	bb := model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.Center, w)

	// with background, no margin, with border, without border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJRound
	td.Dy = -bb.Height() / 2
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.Left, w)

	// with background, no margin, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJRound
	td.Dy = -bb.Height() / 2
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.Center, w)

	// with background, with margins, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJRound
	td.Dy = 0
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.BottomLeft, w)

	// with background, with margins, show margins, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, true
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = color.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = types.LJRound
	td.Dy = 0
	model.WriteColumnAnchored(xRefTable, p.Buf, mediaBox, region, td, types.BottomCenter, w)

	draw.DrawHairCross(p.Buf, 0, 0, r)

	return p
}

func createTextBorderTest(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	var region *types.Rectangle
	writeTextBorderTest(xRefTable, p, region)
	region = types.RectForWidthAndHeight(70, 200, 200, 200)
	writeTextBorderTest(xRefTable, p, region)
	return p
}

func createTextBorderNoMarginAlignLeftTest(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		BorderCol:      color.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 100, 450, types.AlignLeft, types.AlignTop
	td.MinHeight = 300
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 400)

	draw.SetLineWidth(p.Buf, 0)
	draw.SetStrokeColor(p.Buf, color.Black)
	draw.DrawLineSimple(p.Buf, 100, 0, 100, 600)
	draw.DrawLineSimple(p.Buf, 500, 0, 500, 600)
	draw.DrawLineSimple(p.Buf, 110, 0, 110, 600)
	draw.DrawLineSimple(p.Buf, 490, 0, 490, 600)
	draw.DrawLineSimple(p.Buf, 0, 150, 600, 150)
	draw.DrawLineSimple(p.Buf, 0, 450, 600, 450)
	draw.DrawLineSimple(p.Buf, 0, 160, 600, 160)
	draw.DrawLineSimple(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignRightTest(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		BorderCol:      color.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 500, 450, types.AlignRight, types.AlignTop
	td.MinHeight = 300
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 400)

	draw.SetLineWidth(p.Buf, 0)
	draw.SetStrokeColor(p.Buf, color.Black)
	draw.DrawLineSimple(p.Buf, 100, 0, 100, 600)
	draw.DrawLineSimple(p.Buf, 500, 0, 500, 600)
	draw.DrawLineSimple(p.Buf, 110, 0, 110, 600)
	draw.DrawLineSimple(p.Buf, 490, 0, 490, 600)
	draw.DrawLineSimple(p.Buf, 0, 150, 600, 150)
	draw.DrawLineSimple(p.Buf, 0, 450, 600, 450)
	draw.DrawLineSimple(p.Buf, 0, 160, 600, 160)
	draw.DrawLineSimple(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignCenterTest(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		BorderCol:      color.NewSimpleColor(0xabe003),
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 300, 450, types.AlignCenter, types.AlignTop
	td.MinHeight = 300
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 400)

	draw.SetLineWidth(p.Buf, 0)
	draw.SetStrokeColor(p.Buf, color.Black)
	draw.DrawLineSimple(p.Buf, 100, 0, 100, 600)
	draw.DrawLineSimple(p.Buf, 500, 0, 500, 600)
	draw.DrawLineSimple(p.Buf, 110, 0, 110, 600)
	draw.DrawLineSimple(p.Buf, 490, 0, 490, 600)
	draw.DrawLineSimple(p.Buf, 0, 150, 600, 150)
	draw.DrawLineSimple(p.Buf, 0, 450, 600, 450)
	draw.DrawLineSimple(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignJustifyTest(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		BorderCol:      color.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 100, 450, types.AlignJustify, types.AlignTop
	td.MinHeight = 300
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 400)

	draw.SetLineWidth(p.Buf, 0)
	draw.SetStrokeColor(p.Buf, color.Black)
	draw.DrawLineSimple(p.Buf, 100, 0, 100, 600)
	draw.DrawLineSimple(p.Buf, 500, 0, 500, 600)
	draw.DrawLineSimple(p.Buf, 110, 0, 110, 600)
	draw.DrawLineSimple(p.Buf, 490, 0, 490, 600)
	draw.DrawLineSimple(p.Buf, 0, 150, 600, 150)
	draw.DrawLineSimple(p.Buf, 0, 450, 600, 450)
	draw.DrawLineSimple(p.Buf, 0, 160, 600, 160)
	draw.DrawLineSimple(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createXRefAndWritePDF(t *testing.T, msg, fileName string, mediaBox *types.Rectangle, f func(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page) {
	t.Helper()
	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	p := f(xRefTable, mediaBox)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	outDir := filepath.Join("..", "..", "samples", "basic")
	outFile := filepath.Join(outDir, fileName+".pdf")
	createAndValidate(t, xRefTable, outFile, msg)
}

func testTextDemoPDF(t *testing.T, msg, fileName string, w, h int, hAlign types.HAlignment) {
	t.Helper()

	var f1, f2, f3, f4 func(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page

	switch hAlign {
	case types.AlignLeft:
		f1 = createTextDemoAlignLeft
		f2 = createTextDemoAlignLeftMargin
		f3 = createTextDemoAlignLeftWidth
		f4 = createTextDemoAlignLeftWidthAndMargin
	case types.AlignCenter:
		f1 = createTextDemoAlignCenter
		f2 = createTextDemoAlignCenterMargin
		f3 = createTextDemoAlignCenterWidth
		f4 = createTextDemoAlignCenterWidthAndMargin
	case types.AlignRight:
		f1 = createTextDemoAlignRight
		f2 = createTextDemoAlignRightMargin
		f3 = createTextDemoAlignRightWidth
		f4 = createTextDemoAlignRightWidthAndMargin
	case types.AlignJustify:
		f1 = createTextDemoAlignJustify
		f2 = createTextDemoAlignJustifyMargin
		f3 = createTextDemoAlignJustifyWidth
		f4 = createTextDemoAlignJustifyWidthAndMargin
	}

	mediaBox := types.RectForDim(float64(w), float64(h))
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName, mediaBox, f1)
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"Margin", mediaBox, f2)
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"Width", mediaBox, f3)
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"WidthAndMargin", mediaBox, f4)
}

func TestTextDemoPDF(t *testing.T) {
	msg := "TestTextDemoPDF"
	w, h := 600, 600

	for _, tt := range []struct {
		fileName string
		w, h     int
		hAlign   types.HAlignment
	}{
		{"AlignLeft", w, h, types.AlignLeft},
		{"AlignCenter", w, h, types.AlignCenter},
		{"AlignRight", w, h, types.AlignRight},
		{"AlignJustify", w, h, types.AlignJustify},
	} {
		testTextDemoPDF(t, msg, tt.fileName, tt.w, tt.h, tt.hAlign)
	}
}

func TestColumnDemoPDF(t *testing.T) {
	msg := "TestColumnDemoPDF"

	for _, tt := range []struct {
		fileName string
		w, h     int
		f        func(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page
	}{
		{"TestTextAlignJustifyDemo", 600, 600, createTextAlignJustifyDemo},
		{"TestTextAlignJustifyColumnDemo", 600, 600, createTextAlignJustifyColumnDemo},
		{"TextDemoAnchors", 600, 600, createTextDemoAnchors},
		{"TextDemoAnchorsWithOffset", 600, 600, createTextDemoAnchorsWithOffset},
		{"TextDemoColumnAnchored", 1200, 1200, createTextDemoColumnAnchored},
		{"TextDemoColumnAnchoredWithOffset", 1200, 1200, createTextDemoColumnAnchoredWithOffset},
		{"TextRotateDemo", 1200, 1200, createTextRotateDemo},
		{"TextRotateDemoWithOffset", 1200, 1200, createTextRotateDemoWithOffset},
		{"TextScaleAbsoluteDemo", 600, 600, createTextScaleAbsoluteDemo},
		{"TextScaleAbsoluteDemoWithOffset", 600, 600, createTextScaleAbsoluteDemoWithOffset},
		{"TextScaleRelativeDemo", 600, 600, createTextScaleRelativeDemo},
		{"TextScaleRelativeDemoWithOffset", 600, 600, createTextScaleRelativeDemoWithOffset},
		{"TextDemoColumns", 600, 600, createTextDemoColumns},
		{"TextBorderTest", 600, 600, createTextBorderTest},
		{"TextBorderNoMarginAlignLeftTest", 600, 600, createTextBorderNoMarginAlignLeftTest},
		{"TextBorderNoMarginAlignRightTest", 600, 600, createTextBorderNoMarginAlignRightTest},
		{"TextBorderNoMarginAlignCenterTest", 600, 600, createTextBorderNoMarginAlignCenterTest},
		{"TextBorderNoMarginAlignJustifyTest", 600, 600, createTextBorderNoMarginAlignJustifyTest},
	} {
		mediaBox := types.RectForDim(float64(tt.w), float64(tt.h))
		createXRefAndWritePDF(t, msg, tt.fileName, mediaBox, tt.f)
	}
}

func writecreateTestRTLUserFont(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, fontName, s string) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           s,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		RTL:            true,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		X:              mediaBox.Width(),
		Y:              -1,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         types.AlignRight,
		VAlign:         types.AlignMiddle,
		RMode:          draw.RMFill,
		StrokeCol:      color.NewSimpleColor(0x206A29),
		FillCol:        color.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)

	draw.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTestRTLUserFont(xRefTable *model.XRefTable, mediaBox *types.Rectangle, language, fontName string) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	text := sampleTextRTL[language]
	writecreateTestRTLUserFont(xRefTable, p, region, fontName, text)
	region = types.RectForWidthAndHeight(10, 10, mediaBox.Width()/4, mediaBox.Height()/4)
	writecreateTestRTLUserFont(xRefTable, p, region, fontName, text)
	return p
}

func writecreateTestUserFontJustified(xRefTable *model.XRefTable, p model.Page, region *types.Rectangle, rtl bool) {
	mediaBox := p.MediaBox
	buf := p.Buf

	mediaBB := true

	var cr, cg, cb float32
	cr, cg, cb = .5, .75, 1.
	r := mediaBox
	if region != nil {
		r = region
		cr, cg, cb = .75, .75, 1
	}
	if mediaBB {
		draw.FillRectNoBorder(buf, r, color.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Roboto-Regular"
	k := p.Fm.EnsureKey(fontName)

	td := model.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		Embed:          true,
		FontKey:        k,
		FontSize:       12,
		RTL:            rtl,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		X:              -1,
		Y:              -1,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         types.AlignJustify,
		VAlign:         types.AlignMiddle,
		RMode:          draw.RMFill,
		StrokeCol:      color.NewSimpleColor(0x206A29),
		FillCol:        color.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	model.WriteMultiLine(xRefTable, buf, mediaBox, region, td)

	draw.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTestUserFontJustified(xRefTable *model.XRefTable, mediaBox *types.Rectangle, rtl bool) model.Page {
	p := model.NewPage(mediaBox, nil)
	var region *types.Rectangle
	writecreateTestUserFontJustified(xRefTable, p, region, rtl)
	return p
}

func createXRefAndWriteJustifiedPDF(t *testing.T, msg, fileName string, mediaBox *types.Rectangle, rtl bool) {
	t.Helper()
	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	p := createTestUserFontJustified(xRefTable, mediaBox, rtl)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	outDir := filepath.Join("..", "..", "samples", "basic")
	outFile := filepath.Join(outDir, fileName+".pdf")
	createAndValidate(t, xRefTable, outFile, msg)
}

func TestUserFontJustified(t *testing.T) {
	msg := "TestUserFontJustified"
	mediaBox := types.RectForDim(600, 600)
	createXRefAndWriteJustifiedPDF(t, msg, "UserFont_Justified", mediaBox, false)
	createXRefAndWriteJustifiedPDF(t, msg, "UserFont_JustifiedRightToLeft", mediaBox, true)
}

func createXRefAndWriteRTLPDF(t *testing.T,
	msg, fileName string,
	mediaBox *types.Rectangle,
	language, fontName string,
	f func(xRefTable *model.XRefTable, mediaBox *types.Rectangle, language, fontName string) model.Page) {
	t.Helper()

	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	p := f(xRefTable, mediaBox, language, fontName)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	outDir := filepath.Join("..", "..", "samples", "basic")
	outFile := filepath.Join(outDir, fileName+".pdf")
	createAndValidate(t, xRefTable, outFile, msg)
}

func TestUserFontRTL(t *testing.T) {
	msg := "TestUserFontRTL"
	f := createTestRTLUserFont
	mediaBox := types.RectForDim(600, 600)

	for _, tt := range []struct {
		fileName string
		language string
		fontName string
	}{
		{"UserFont_Arabic", "Arabic", "UnifontMedium"},
		{"UserFont_Hebrew", "Hebrew", "UnifontMedium"},
		{"UserFont_Persian", "Persian", "UnifontMedium"},
		{"UserFont_Urdu", "Urdu", "UnifontMedium"},
	} {
		createXRefAndWriteRTLPDF(t, msg, tt.fileName, mediaBox, tt.language, tt.fontName, f)
	}
}

func createCJKVDemo(xRefTable *model.XRefTable, mediaBox *types.Rectangle) model.Page {
	p := model.NewPage(mediaBox, nil)
	mb := p.MediaBox

	textEnglish := `pdfcpu
Instant PDF processing for all your needs.
Now supporting CJKV!`

	textChineseSimple := `pdfcpu
即时处理PDF，满足您的所有需求。
现在支持CJKV字体！`

	textJapanese := `pdfcpu
すべてのニーズに対応するインスタントPDF処理。
CJKVフォントがサポートされるようになりました！`

	textKorean := `pdfcpu
모든 요구 사항에 맞는 즉각적인 PDF 처리.
이제 CJKV 글꼴을 지원합니다!`

	textVietnamese := `pdfcpu
Xử lý PDF tức thì cho mọi nhu cầu của bạn.
Bây giờ với sự hỗ trợ cho các phông chữ CJKV!`

	td := model.TextDescriptor{
		FontSize:       24,
		Embed:          true,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1,
		ScaleAbs:       true,
		HAlign:         types.AlignLeft,
		VAlign:         types.AlignMiddle,
		RMode:          draw.RMFill,
		StrokeCol:      color.NewSimpleColor(0x206A29),
		FillCol:        color.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Text, td.FontName, td.FontKey = textChineseSimple, "UnifontMedium", p.Fm.EnsureKey("UnifontMedium")
	td.X, td.Y = 0, mb.Height()
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textJapanese, "Unifont-JPMedium", p.Fm.EnsureKey("Unifont-JPMedium")
	td.X, td.Y = mb.Width(), 2*mb.Height()/3
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textKorean, "UnifontMedium", p.Fm.EnsureKey("UnifontMedium")
	td.X, td.Y = 0, mb.Height()/3
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textVietnamese, "Roboto-Regular", p.Fm.EnsureKey("Roboto-Regular")
	td.X, td.Y = mb.Width(), 0
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontSize, td.ShowTextBB = textEnglish, 24, false
	td.X, td.Y, td.HAlign = -1, -1, types.AlignCenter
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 0)

	td.FontSize = 80
	td.Text, td.HAlign, td.X, td.Y = "C", types.AlignRight, mb.Width(), mb.Height()
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "J", types.AlignLeft, 0, 2*mb.Height()/3
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "K", types.AlignRight, mb.Width(), mb.Height()/3
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "V", types.AlignLeft, 0, 0
	model.WriteColumn(xRefTable, p.Buf, mediaBox, nil, td, 0)

	return p
}

func TestCJKV(t *testing.T) {
	msg := "TestCJKV"
	mediaBox := types.RectForDim(600, 600)
	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	p := createCJKVDemo(xRefTable, mediaBox)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	outDir := filepath.Join("..", "..", "samples", "basic")
	outFile := filepath.Join(outDir, "UserFont_CJKV.pdf")
	createAndValidate(t, xRefTable, outFile, msg)
}
