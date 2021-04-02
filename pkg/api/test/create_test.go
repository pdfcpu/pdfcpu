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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
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

func createAndValidate(t *testing.T, xRefTable *pdf.XRefTable, outFile, msg string) {
	t.Helper()
	outDir := "../../samples/create"
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
	mediaBox := pdf.RectForFormat("A4")
	p := pdf.Page{MediaBox: mediaBox, Fm: pdf.FontMap{}, Buf: new(bytes.Buffer)}
	pdf.CreateTestPageContent(p)
	xRefTable, err := pdf.CreateDemoXRef(p)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "Test.pdf", msg)
}

func TestResourceDictInheritanceDemoPDF(t *testing.T) {
	// Create a test page proofing resource inheritance.
	// Resources may be inherited from ANY parent node.
	// Case in point: fonts
	msg := "TestResourceDictInheritanceDemoPDF"
	xRefTable, err := pdf.CreateResourceDictInheritanceDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "ResourceDictInheritanceDemo.pdf", msg)
}

func TestAnnotationDemoPDF(t *testing.T) {
	msg := "TestAnnotationDemoPDF"
	xRefTable, err := pdf.CreateAnnotationDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "AnnotationDemo.pdf", msg)
}

func TestAcroformDemoPDF(t *testing.T) {
	msg := "TestAcroformDemoPDF"
	xRefTable, err := pdf.CreateAcroFormDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, "AcroFormDemo.pdf", msg)
}

func writeTextDemoAlignedWidthAndMargin(
	p pdf.Page,
	region *pdf.Rectangle,
	hAlign pdf.HAlignment,
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		FontName:       fontName,
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
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		FillCol:        pdf.Black,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     true,
		ShowTextBB:     true,
		HairCross:      true,
	}

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBaseline, -1, r.Height()*.75, "M\\u(lti\nline\n\nwith empty line"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBaseline, r.Width()*.75, r.Height()*.25, "Arbitrary\ntext\nlines"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	// Multilines along the top of the page:
	td.VAlign, td.X, td.Y, td.Text = pdf.AlignTop, 0, r.Height(), "0,h (topleft)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignTop, -1, r.Height(), "-1,h (topcenter)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignTop, r.Width(), r.Height(), "w,h (topright)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	// Multilines along the center of the page:
	// x = 0 centers the position of multilines horizontally
	// y = 0 centers the position of multilines vertically and enforces alignMiddle
	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBaseline, 0, -1, "0,-1 (left)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignMiddle, -1, -1, "-1,-1 (center)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBaseline, r.Width(), -1, "w,-1 (right)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	// Multilines along the bottom of the page:
	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBottom, 0, 0, "0,0 (botleft)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBottom, -1, 0, "-1,0 (botcenter)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	td.VAlign, td.X, td.Y, td.Text = pdf.AlignBottom, r.Width(), 0, "w,0 (botright)\nand line2"
	pdf.WriteColumn(buf, mediaBox, region, td, w)

	pdf.DrawHairCross(buf, 0, 0, r)
}

func createTextDemoAlignedWidthAndMargin(mediaBox *pdf.Rectangle, hAlign pdf.HAlignment, w, mLeft, mRight, mTop, mBot float64) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextDemoAlignedWidthAndMargin(p, region, hAlign, w, mLeft, mRight, mTop, mBot)
	region = pdf.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAlignedWidthAndMargin(p, region, hAlign, w, mLeft, mRight, mTop, mBot)
	return p
}

func createTextDemoAlignLeft(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignLeft, 0, 0, 0, 0, 0)
}

func createTextDemoAlignLeftMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignLeft, 0, 5, 10, 15, 20)
}

func createTextDemoAlignRight(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignRight, 0, 0, 0, 0, 0)
}

func createTextDemoAlignRightMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignRight, 0, 5, 10, 15, 20)
}

func createTextDemoAlignCenter(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignCenter, 0, 0, 0, 0, 0)
}

func createTextDemoAlignCenterMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignCenter, 0, 5, 10, 15, 20)
}

func createTextDemoAlignJustify(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignJustify, 0, 0, 0, 0, 0)
}

func createTextDemoAlignJustifyMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignJustify, 0, 5, 10, 15, 20)
}

func createTextDemoAlignLeftWidth(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignLeft, 250, 0, 0, 0, 0)
}

func createTextDemoAlignLeftWidthAndMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignLeft, 250, 5, 10, 15, 20)
}

func createTextDemoAlignRightWidth(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignRight, 250, 0, 0, 0, 0)
}

func createTextDemoAlignRightWidthAndMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignRight, 250, 5, 10, 15, 20)
}

func createTextDemoAlignCenterWidth(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignCenter, 250, 0, 0, 0, 0)
}

func createTextDemoAlignCenterWidthAndMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignCenter, 250, 5, 40, 15, 20)
}

func createTextDemoAlignJustifyWidth(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignJustify, 250, 0, 0, 0, 0)
}

func createTextDemoAlignJustifyWidthAndMargin(mediaBox *pdf.Rectangle) pdf.Page {
	return createTextDemoAlignedWidthAndMargin(mediaBox, pdf.AlignJustify, 250, 5, 10, 15, 20)
}

func writeTextAlignJustifyDemo(p pdf.Page, region *pdf.Rectangle, fontName string) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
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
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	pdf.WriteMultiLine(buf, mediaBox, region, td)

	pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func writeTextAlignJustifyColumnDemo(p pdf.Page, region *pdf.Rectangle) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Times-Roman"
	fontName2 := "Helvetica"
	k1 := p.Fm.EnsureKey(fontName)
	k2 := p.Fm.EnsureKey(fontName2)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1.,
		ScaleAbs:       true,
		HAlign:         pdf.AlignJustify,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		FillCol:        pdf.Black,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.BackgroundCol = pdf.White
	td.FillCol = pdf.Black
	td.FontName, td.FontKey, td.FontSize = fontName, k1, 9
	td.ParIndent = true
	td.VAlign, td.X, td.Y, td.Dx, td.Dy = pdf.AlignTop, 0, r.Height(), 5, -5
	pdf.WriteColumn(buf, mediaBox, region, td, 150)

	td.BackgroundCol = pdf.Black
	td.FillCol = pdf.White
	td.FontName, td.FontKey, td.FontSize = fontName2, k2, 12
	td.ParIndent = true
	td.VAlign, td.X, td.Y, td.Dx, td.Dy = pdf.AlignTop, -1, -1, 0, 0
	pdf.WriteColumn(buf, mediaBox, region, td, 290)

	pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTextAlignJustifyDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	fontName := "Times-Roman"
	writeTextAlignJustifyDemo(p, region, fontName)
	region = pdf.RectForWidthAndHeight(0, 0, 200, 200)
	writeTextAlignJustifyDemo(p, region, fontName)
	return p
}

func createTextAlignJustifyColumnDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextAlignJustifyColumnDemo(p, region)
	region = pdf.RectForWidthAndHeight(0, 0, 200, 200)
	writeTextAlignJustifyColumnDemo(p, region)
	return p
}

func writeTextDemoAnchorsWithOffset(p pdf.Page, region *pdf.Rectangle, dx, dy float64) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		FontName:       fontName,
		FontKey:        k,
		FontSize:       24,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		FillCol:        pdf.Black,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     true,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy, td.Text = dx, -dy, "topleft\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopLeft)

	td.Dx, td.Dy, td.Text = 0, -dy, "topcenter\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopCenter)

	td.Dx, td.Dy, td.Text = -dx, -dy, "topright\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopRight)

	td.Dx, td.Dy, td.Text = dx, 0, "left\nandline2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Left)

	td.Dx, td.Dy, td.Text = 0, 0, "center\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Center)

	td.Dx, td.Dy, td.Text = -dx, 0, "right\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Right)

	td.Dx, td.Dy, td.Text = dx, dy, "botleft\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomLeft)

	td.Dx, td.Dy, td.Text = 0, dy, "botcenter\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomCenter)

	td.Dx, td.Dy, td.Text = -dx, dy, "botright\nandLine2"
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomRight)

	pdf.DrawHairCross(buf, 0, 0, r)
}

func writeTextDemoAnchors(p pdf.Page, region *pdf.Rectangle) {
	writeTextDemoAnchorsWithOffset(p, region, 0, 0)
}

func createTextDemoAnchors(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextDemoAnchors(p, region)
	region = pdf.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAnchors(p, region)
	return p
}

func createTextDemoAnchorsWithOffset(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	dx, dy := 20., 20.
	var region *pdf.Rectangle
	writeTextDemoAnchorsWithOffset(p, region, dx, dy)
	region = pdf.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextDemoAnchorsWithOffset(p, region, dx, dy)
	return p
}

func writeTextDemoColumnAnchoredWithOffset(p pdf.Page, region *pdf.Rectangle, dx, dy float64) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	wSmall := 100.
	wBig := 300.

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       6,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		FillCol:        pdf.Black,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy = dx, -dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopLeft, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopLeft, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopLeft, wBig)

	td.Dx, td.Dy = 0, -dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopCenter, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopCenter, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopCenter, wBig)

	td.Dx, td.Dy = -dx, -dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopRight, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopRight, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopRight, wBig)

	td.Dx, td.Dy = dx, 0
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Left, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Left, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Left, wBig)

	td.Dx, td.Dy = 0, 0
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Center, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Center, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Center, wBig)

	td.Dx, td.Dy = -dx, 0
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Right, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Right, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.Right, wBig)

	td.Dx, td.Dy = dx, dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomLeft, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomLeft, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomLeft, wBig)

	td.Dx, td.Dy = 0, dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomCenter, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomCenter, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomCenter, wBig)

	td.Dx, td.Dy = -dx, dy
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomRight, wSmall)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomRight, 0)
	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.BottomRight, wBig)

	pdf.DrawHairCross(buf, 0, 0, mediaBox)
}

func writeTextDemoColumnAnchored(p pdf.Page, region *pdf.Rectangle) {
	writeTextDemoColumnAnchoredWithOffset(p, region, 0, 0)
}

func createTextDemoColumnAnchored(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextDemoColumnAnchored(p, region)
	region = pdf.RectForWidthAndHeight(50, 70, 400, 400)
	writeTextDemoColumnAnchored(p, region)
	return p
}

func createTextDemoColumnAnchoredWithOffset(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	dx, dy := 20., 20.
	writeTextDemoColumnAnchoredWithOffset(p, region, dx, dy)
	region = pdf.RectForWidthAndHeight(50, 70, 400, 400)
	writeTextDemoColumnAnchoredWithOffset(p, region, dx, dy)
	return p
}

func writeTextRotateDemoWithOffset(p pdf.Page, region *pdf.Rectangle, dx, dy float64) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
		pdf.DrawHairCross(buf, 0, 0, r)
	}

	fillCol := pdf.Black

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           "Hello Gopher!\nLine 2",
		FontName:       fontName,
		FontKey:        k,
		FontSize:       24,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Dx, td.Dy = dx, -dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{R: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{G: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{G: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{B: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{B: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{R: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)

	td.Dx, td.Dy = 0, 0
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{G: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{G: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)

	td.Dx, td.Dy = -dx, 0
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{B: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{B: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)

	td.Dx, td.Dy = dx, dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{R: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{G: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{G: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)

	td.Dx, td.Dy = -dx, dy
	td.Rotation, td.FillCol = 0, fillCol
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)
	td.Rotation, td.FillCol = 45, pdf.SimpleColor{B: 1}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)
	td.Rotation, td.FillCol = 90, pdf.SimpleColor{B: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)
}

func writeTextRotateDemo(p pdf.Page, region *pdf.Rectangle) {
	writeTextRotateDemoWithOffset(p, region, 0, 0)
}

func createTextRotateDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextRotateDemo(p, region)
	region = pdf.RectForWidthAndHeight(150, 150, 300, 300)
	writeTextRotateDemo(p, region)
	return p
}

func createTextRotateDemoWithOffset(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	dx, dy := 20., 20.
	writeTextRotateDemoWithOffset(p, region, dx, dy)
	region = pdf.RectForWidthAndHeight(150, 150, 300, 300)
	writeTextRotateDemoWithOffset(p, region, dx, dy)
	return p
}

func writeTextScaleAbsoluteDemoWithOffset(p pdf.Page, region *pdf.Rectangle, dx, dy float64) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fillCol := pdf.Black
	bgCol := pdf.SimpleColor{R: 1., G: .98, B: .77}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.HAlign, td.VAlign, td.X, td.Y, td.FontSize = pdf.AlignJustify, pdf.AlignMiddle, -1, r.Height()*.72, 9
	td.Scale, td.FillCol = 1, fillCol
	pdf.WriteMultiLine(buf, mediaBox, region, td)
	td.Scale, td.FillCol = 1.5, pdf.SimpleColor{R: 1}
	pdf.WriteMultiLine(buf, mediaBox, region, td)
	td.Scale, td.FillCol = 2, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLine(buf, mediaBox, region, td)

	width := 130.

	td.HAlign, td.VAlign, td.X = pdf.AlignJustify, pdf.AlignMiddle, r.Width()*.75
	td.FillCol, td.Text = fillCol, "Justified column\nWidth=130"

	td.FontSize, td.Y = 24, r.Height()*.35
	td.Scale = 1
	pdf.WriteColumn(buf, mediaBox, region, td, width)
	td.Scale = 1.5
	pdf.WriteColumn(buf, mediaBox, region, td, width)

	td.FontSize, td.Y = 12, r.Height()*.22
	td.Scale = 1
	pdf.WriteColumn(buf, mediaBox, region, td, width)
	td.Scale = 1.5
	pdf.WriteColumn(buf, mediaBox, region, td, width)

	td.FontSize = 9
	td.Scale, td.Y = 1, r.Height()*.15
	pdf.WriteColumn(buf, mediaBox, region, td, width)
	td.Scale, td.Y = 1.5, r.Height()*.13
	pdf.WriteColumn(buf, mediaBox, region, td, width)

	td = pdf.TextDescriptor{
		FontName:       fontName,
		FontKey:        k,
		FontSize:       12,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
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
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{R: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{R: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{G: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{G: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{B: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{B: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{R: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)
	td.Scale, td.FillCol = .5, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Left)

	td.Dx, td.Dy = 0, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{G: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{G: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Center)

	td.Dx, td.Dy = -dx, 0
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{B: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{B: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.Right)

	td.Dx, td.Dy = dx, dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{R: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)
	td.Scale, td.FillCol = .5, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{G: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{G: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomCenter)

	td.Dx, td.Dy = -dx, +dy
	td.Scale, td.FillCol, td.Text = 1.5, fillCol, text15
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)
	td.Scale, td.FillCol, td.Text = 1, pdf.SimpleColor{B: 1}, text1
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)
	td.Scale, td.FillCol, td.Text = .5, pdf.SimpleColor{B: .5}, text5
	pdf.WriteMultiLineAnchored(buf, mediaBox, r, td, pdf.BottomRight)

	pdf.DrawHairCross(buf, 0, 0, r)
}

func writeTextScaleAbsoluteDemo(p pdf.Page, region *pdf.Rectangle) {
	writeTextScaleAbsoluteDemoWithOffset(p, region, 0, 0)
}

func createTextScaleAbsoluteDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextScaleAbsoluteDemo(p, region)
	region = pdf.RectForWidthAndHeight(20, 70, 180, 180)
	writeTextScaleAbsoluteDemo(p, region)
	return p
}

func createTextScaleAbsoluteDemoWithOffset(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	dx, dy := 20., 20.
	var region *pdf.Rectangle
	writeTextScaleAbsoluteDemoWithOffset(p, region, dx, dy)
	region = pdf.RectForWidthAndHeight(20, 70, 180, 180)
	writeTextScaleAbsoluteDemoWithOffset(p, region, dx, dy)
	return p
}

func writeTextScaleRelativeDemoWithOffset(p pdf.Page, region *pdf.Rectangle, dx, dy float64) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fillCol := pdf.Black
	bgCol := pdf.SimpleColor{R: 1., G: .98, B: .77}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		X:              -1,
		Y:              r.Height() * .73,
		ScaleAbs:       false,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.FontSize, td.Scale, td.FillCol = 9, .4, fillCol
	pdf.WriteMultiLine(buf, mediaBox, region, td)
	td.FontSize, td.Scale, td.FillCol = 9, .6, pdf.SimpleColor{R: 1}
	pdf.WriteMultiLine(buf, mediaBox, region, td)
	td.FontSize, td.Scale, td.FillCol = 9, .8, pdf.SimpleColor{R: .5}
	pdf.WriteMultiLine(buf, mediaBox, region, td)

	width := 130.

	td = pdf.TextDescriptor{
		Text:           "Justified column\nWidth=130",
		FontName:       fontName,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		X:              r.Width() * .75,
		Y:              r.Height() * .25,
		ScaleAbs:       false,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		ShowBackground: true,
		BackgroundCol:  bgCol,
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}
	td.Scale, td.FillCol = .5, fillCol
	pdf.WriteColumn(buf, mediaBox, region, td, width)
	td.Scale, td.FillCol = .3, pdf.SimpleColor{G: 1}
	pdf.WriteColumn(buf, mediaBox, region, td, width)
	td.Scale, td.FillCol = .20, pdf.SimpleColor{G: .5}
	pdf.WriteColumn(buf, mediaBox, region, td, width)

	td = pdf.TextDescriptor{
		FontName:       fontName,
		FontKey:        k,
		FontSize:       18,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		X:              r.Width() * .75,
		Y:              r.Height() * .25,
		ScaleAbs:       false,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
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
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopLeft)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{R: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopLeft)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{R: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopLeft)

	td.Dx, td.Dy = 0, -dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopCenter)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{G: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopCenter)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{G: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopCenter)

	td.Dx, td.Dy = -dx, -dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopRight)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{B: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopRight)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{B: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.TopRight)

	td.Dx, td.Dy = dx, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Left)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{R: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Left)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{R: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Left)

	td.Dx, td.Dy = 0, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Center)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{G: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Center)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{G: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Center)

	td.Dx, td.Dy = -dx, 0
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Right)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{B: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Right)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{B: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.Right)

	td.Dx, td.Dy = dx, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomLeft)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{R: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomLeft)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{R: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomLeft)

	td.Dx, td.Dy = 0, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomCenter)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{G: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomCenter)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{G: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomCenter)

	td.Dx, td.Dy = -dx, dy
	td.Scale, td.FillCol, td.Text = .3, fillCol, text30
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomRight)
	td.Scale, td.FillCol, td.Text = .2, pdf.SimpleColor{B: 1}, text20
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomRight)
	td.Scale, td.FillCol, td.Text = .1, pdf.SimpleColor{B: .5}, text10
	pdf.WriteMultiLineAnchored(buf, mediaBox, region, td, pdf.BottomRight)

	pdf.DrawHairCross(buf, 0, 0, r)
}

func writeTextScaleRelativeDemo(p pdf.Page, region *pdf.Rectangle) {
	writeTextScaleRelativeDemoWithOffset(p, region, 0, 0)
}

func createTextScaleRelativeDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writeTextScaleRelativeDemo(p, region)
	region = pdf.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextScaleRelativeDemo(p, region)
	return p
}

func createTextScaleRelativeDemoWithOffset(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	dx, dy := 20., 20.
	writeTextScaleRelativeDemoWithOffset(p, region, dx, dy)
	region = pdf.RectForWidthAndHeight(50, 70, 200, 200)
	writeTextScaleRelativeDemoWithOffset(p, region, dx, dy)
	return p
}

func createTextDemoColumns(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		FontName:       fontName,
		FontKey:        k,
		FontSize:       9,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
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
	td.BackgroundCol = pdf.SimpleColor{R: .4, G: .98, B: .77}
	td.BorderStyle = pdf.LJRound
	pdf.WriteColumnAnchored(p.Buf, mediaBox, nil, td, pdf.TopLeft, width)

	// Render middle column.
	// Draw the bounding box with regular corners but no border.
	td.Text = sampleText2
	td.Dx = -width / 2
	td.ShowTextBB, td.ShowBorder = true, false
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJMiter
	pdf.WriteColumnAnchored(p.Buf, mediaBox, nil, td, pdf.TopCenter, width)

	// Render right column.
	// Draw bounding box and a border with rounded corners.

	td.Text = sampleText3
	td.Dx = 0
	td.ShowTextBB, td.ShowBorder = true, true
	td.BackgroundCol = pdf.SimpleColor{R: 1., G: .98, B: .77}
	td.BorderCol = pdf.SimpleColor{R: .2, G: .5, B: .2}
	td.BorderStyle = pdf.LJRound
	pdf.WriteColumnAnchored(p.Buf, mediaBox, nil, td, pdf.TopRight, width)

	// 2nd row: 3 side by side columns below using relative scaling,
	// Indent paragraph beginnings and don't draw the background.
	relScaleFactor := .334
	td.Dy = mediaBox.Height() / 2
	td.Scale = relScaleFactor
	td.ScaleAbs = false
	td.ParIndent = true
	td.ShowBackground, td.ShowBorder = false, true
	td.HAlign, td.VAlign = pdf.AlignJustify, pdf.AlignTop

	// Render left column.
	td.Text = sampleText
	td.X = 0
	td.ShowTextBB = true
	td.BorderStyle = pdf.LJBevel
	pdf.WriteMultiLine(p.Buf, mediaBox, nil, td)

	// Render middle column.
	td.Text = sampleText2
	td.X = mediaBox.Width() / 2
	td.ShowTextBB = false
	pdf.WriteMultiLine(p.Buf, mediaBox, nil, td)

	// Render right column.
	td.Text = sampleText3
	td.X = mediaBox.Width()
	td.Dx = 0
	td.ShowTextBB = true
	td.BorderStyle = pdf.LJMiter
	pdf.WriteMultiLine(p.Buf, mediaBox, nil, td)

	pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)

	return p
}

func writeTextBorderTest(p pdf.Page, region *pdf.Rectangle) pdf.Page {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		FontName:   fontName,
		FontKey:    k,
		FontSize:   7,
		MLeft:      10,
		MRight:     10,
		MTop:       10,
		MBot:       10,
		Scale:      1.,
		ScaleAbs:   true,
		RMode:      pdf.RMFill,
		BorderCol:  pdf.NewSimpleColor(0xabe003),
		ShowTextBB: true,
	}

	w := mediaBox.Width() / 2

	// no background, no margin, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = false, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BorderWidth = 0
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJMiter
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.TopLeft, w)

	// with background, no margin, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BorderWidth = 0
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.TopCenter, w)

	// with background, with margins, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.Dy = 100
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.Left, w)

	// with background, with margins, show margins, no border
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, true
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJMiter
	td.Dy = 100
	bb := pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.Center, w)

	// with background, no margin, with border, without border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, false, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJRound
	td.Dy = -bb.Height() / 2
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.Left, w)

	// with background, no margin, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 0, 0, 0, 0
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJRound
	td.Dy = -bb.Height() / 2
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.Center, w)

	// with background, with margins, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, false
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJRound
	td.Dy = 0
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.BottomLeft, w)

	// with background, with margins, show margins, with border, with border background
	td.Text = sampleText2
	td.ShowBackground, td.ShowBorder, td.ShowMargins = true, true, true
	td.BorderWidth = 5
	td.MBot, td.MTop, td.MLeft, td.MRight = 10, 10, 10, 10
	td.BackgroundCol = pdf.SimpleColor{R: .6, G: .98, B: .77}
	td.BorderStyle = pdf.LJRound
	td.Dy = 0
	pdf.WriteColumnAnchored(p.Buf, mediaBox, region, td, pdf.BottomCenter, w)

	pdf.DrawHairCross(p.Buf, 0, 0, r)

	return p
}

func createTextBorderTest(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	var region *pdf.Rectangle
	writeTextBorderTest(p, region)
	region = pdf.RectForWidthAndHeight(70, 200, 200, 200)
	writeTextBorderTest(p, region)
	return p
}

func createTextBorderNoMarginAlignLeftTest(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		BorderCol:      pdf.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 100, 450, pdf.AlignLeft, pdf.AlignTop
	td.MinHeight = 300
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 400)

	pdf.SetLineWidth(p.Buf, 0)
	pdf.SetStrokeColor(p.Buf, pdf.Black)
	pdf.DrawLine(p.Buf, 100, 0, 100, 600)
	pdf.DrawLine(p.Buf, 500, 0, 500, 600)
	pdf.DrawLine(p.Buf, 110, 0, 110, 600)
	pdf.DrawLine(p.Buf, 490, 0, 490, 600)
	pdf.DrawLine(p.Buf, 0, 150, 600, 150)
	pdf.DrawLine(p.Buf, 0, 450, 600, 450)
	pdf.DrawLine(p.Buf, 0, 160, 600, 160)
	pdf.DrawLine(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignRightTest(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		BorderCol:      pdf.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 500, 450, pdf.AlignRight, pdf.AlignTop
	td.MinHeight = 300
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 400)

	pdf.SetLineWidth(p.Buf, 0)
	pdf.SetStrokeColor(p.Buf, pdf.Black)
	pdf.DrawLine(p.Buf, 100, 0, 100, 600)
	pdf.DrawLine(p.Buf, 500, 0, 500, 600)
	pdf.DrawLine(p.Buf, 110, 0, 110, 600)
	pdf.DrawLine(p.Buf, 490, 0, 490, 600)
	pdf.DrawLine(p.Buf, 0, 150, 600, 150)
	pdf.DrawLine(p.Buf, 0, 450, 600, 450)
	pdf.DrawLine(p.Buf, 0, 160, 600, 160)
	pdf.DrawLine(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignCenterTest(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		BorderCol:      pdf.NewSimpleColor(0xabe003),
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
		ShowTextBB:     true,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 300, 450, pdf.AlignCenter, pdf.AlignTop
	td.MinHeight = 300
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 400)

	pdf.SetLineWidth(p.Buf, 0)
	pdf.SetStrokeColor(p.Buf, pdf.Black)
	pdf.DrawLine(p.Buf, 100, 0, 100, 600)
	pdf.DrawLine(p.Buf, 500, 0, 500, 600)
	pdf.DrawLine(p.Buf, 110, 0, 110, 600)
	pdf.DrawLine(p.Buf, 490, 0, 490, 600)
	pdf.DrawLine(p.Buf, 0, 150, 600, 150)
	pdf.DrawLine(p.Buf, 0, 450, 600, 450)
	pdf.DrawLine(p.Buf, 0, 160, 600, 160)
	pdf.DrawLine(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createTextBorderNoMarginAlignJustifyTest(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := pdf.TextDescriptor{
		Text:           sampleText2,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       12,
		Scale:          1.,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: .6, G: .98, B: .77},
		ShowBorder:     true,
		BorderWidth:    10,
		BorderCol:      pdf.NewSimpleColor(0xabe003),
		ShowTextBB:     true,
		ShowMargins:    true,
		MLeft:          10,
		MRight:         10,
		MTop:           10,
		MBot:           10,
	}

	td.X, td.Y, td.HAlign, td.VAlign = 100, 450, pdf.AlignJustify, pdf.AlignTop
	td.MinHeight = 300
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 400)

	pdf.SetLineWidth(p.Buf, 0)
	pdf.SetStrokeColor(p.Buf, pdf.Black)
	pdf.DrawLine(p.Buf, 100, 0, 100, 600)
	pdf.DrawLine(p.Buf, 500, 0, 500, 600)
	pdf.DrawLine(p.Buf, 110, 0, 110, 600)
	pdf.DrawLine(p.Buf, 490, 0, 490, 600)
	pdf.DrawLine(p.Buf, 0, 150, 600, 150)
	pdf.DrawLine(p.Buf, 0, 450, 600, 450)
	pdf.DrawLine(p.Buf, 0, 160, 600, 160)
	pdf.DrawLine(p.Buf, 0, 440, 600, 440)
	//pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
	return p
}

func createXRefAndWritePDF(t *testing.T, msg, fileName string, p pdf.Page) {
	t.Helper()
	xRefTable, err := pdf.CreateDemoXRef(p)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	outDir := filepath.Join("..", "..", "samples", "create")
	outFile := filepath.Join(outDir, fileName+".pdf")
	createAndValidate(t, xRefTable, outFile, msg)
}

func testTextDemoPDF(t *testing.T, msg, fileName string, w, h int, hAlign pdf.HAlignment) {
	t.Helper()

	var f1, f2, f3, f4 func(mediaBox *pdf.Rectangle) pdf.Page

	switch hAlign {
	case pdf.AlignLeft:
		f1 = createTextDemoAlignLeft
		f2 = createTextDemoAlignLeftMargin
		f3 = createTextDemoAlignLeftWidth
		f4 = createTextDemoAlignLeftWidthAndMargin
	case pdf.AlignCenter:
		f1 = createTextDemoAlignCenter
		f2 = createTextDemoAlignCenterMargin
		f3 = createTextDemoAlignCenterWidth
		f4 = createTextDemoAlignCenterWidthAndMargin
	case pdf.AlignRight:
		f1 = createTextDemoAlignRight
		f2 = createTextDemoAlignRightMargin
		f3 = createTextDemoAlignRightWidth
		f4 = createTextDemoAlignRightWidthAndMargin
	case pdf.AlignJustify:
		f1 = createTextDemoAlignJustify
		f2 = createTextDemoAlignJustifyMargin
		f3 = createTextDemoAlignJustifyWidth
		f4 = createTextDemoAlignJustifyWidthAndMargin
	}

	mediaBox := pdf.RectForDim(float64(w), float64(h))
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName, f1(mediaBox))
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"Margin", f2(mediaBox))
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"Width", f3(mediaBox))
	createXRefAndWritePDF(t, msg, "TextDemo"+fileName+"WidthAndMargin", f4(mediaBox))
}

func TestTextDemoPDF(t *testing.T) {
	msg := "TestTextDemoPDF"
	w, h := 600, 600

	for _, tt := range []struct {
		fileName string
		w, h     int
		hAlign   pdf.HAlignment
	}{
		{"AlignLeft", w, h, pdf.AlignLeft},
		{"AlignCenter", w, h, pdf.AlignCenter},
		{"AlignRight", w, h, pdf.AlignRight},
		{"AlignJustify", w, h, pdf.AlignJustify},
	} {
		testTextDemoPDF(t, msg, tt.fileName, tt.w, tt.h, tt.hAlign)
	}
}

func TestColumnDemoPDF(t *testing.T) {
	msg := "TestColumnDemoPDF"

	for _, tt := range []struct {
		fileName string
		w, h     int
		f        func(mediaBox *pdf.Rectangle) pdf.Page
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
		mediaBox := pdf.RectForDim(float64(tt.w), float64(tt.h))
		createXRefAndWritePDF(t, msg, tt.fileName, tt.f(mediaBox))
	}
}

func writecreateTestRTLUserFont(p pdf.Page, region *pdf.Rectangle, fontName, text string) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           text,
		FontName:       fontName,
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
		HAlign:         pdf.AlignRight,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	pdf.WriteMultiLine(buf, mediaBox, region, td)

	pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTestRTLUserFont(mediaBox *pdf.Rectangle, language, fontName string) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	text := sampleTextRTL[language]
	writecreateTestRTLUserFont(p, region, fontName, text)
	region = pdf.RectForWidthAndHeight(10, 10, mediaBox.Width()/4, mediaBox.Height()/4)
	writecreateTestRTLUserFont(p, region, fontName, text)
	return p
}

func writecreateTestUserFontJustified(p pdf.Page, region *pdf.Rectangle, rtl bool) {
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
		pdf.FillRect(buf, r, pdf.SimpleColor{R: cr, G: cg, B: cb})
	}

	fontName := "Roboto-Regular"
	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           sampleText,
		FontName:       fontName,
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
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	pdf.WriteMultiLine(buf, mediaBox, region, td)

	pdf.DrawHairCross(p.Buf, 0, 0, mediaBox)
}

func createTestUserFontJustified(mediaBox *pdf.Rectangle, rtl bool) pdf.Page {
	p := pdf.NewPage(mediaBox)
	var region *pdf.Rectangle
	writecreateTestUserFontJustified(p, region, rtl)
	return p
}

func TestUserFontJustified(t *testing.T) {
	msg := "TestUserFontJustified"

	// Install test user fonts (in addition to already installed user fonts)
	// from pkg/testdata/fonts.
	api.LoadConfiguration()
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	mediaBox := pdf.RectForDim(600, 600)
	createXRefAndWritePDF(t, msg, "UserFont_Justified", createTestUserFontJustified(mediaBox, false))
	createXRefAndWritePDF(t, msg, "UserFont_JustifiedRightToLeft", createTestUserFontJustified(mediaBox, true))
	// Justified text for right to left languages seems to be tricky business - the following samples are right aligned:
	createXRefAndWritePDF(t, msg, "UserFont_Arabic", createTestRTLUserFont(mediaBox, "Arabic", "UnifontMedium"))
	createXRefAndWritePDF(t, msg, "UserFont_Hebrew", createTestRTLUserFont(mediaBox, "Hebrew", "UnifontMedium"))
	createXRefAndWritePDF(t, msg, "UserFont_Persian", createTestRTLUserFont(mediaBox, "Persian", "UnifontMedium"))
	createXRefAndWritePDF(t, msg, "UserFont_Urdu", createTestRTLUserFont(mediaBox, "Urdu", "UnifontMedium"))
}

func createCJKVDemo(mediaBox *pdf.Rectangle) pdf.Page {
	p := pdf.NewPage(mediaBox)
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

	td := pdf.TextDescriptor{
		FontSize:       24,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1,
		ScaleAbs:       true,
		HAlign:         pdf.AlignLeft,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	td.Text, td.FontName, td.FontKey = textChineseSimple, "UnifontMedium", p.Fm.EnsureKey("UnifontMedium")
	td.X, td.Y = 0, mb.Height()
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textJapanese, "Unifont-JPMedium", p.Fm.EnsureKey("Unifont-JPMedium")
	td.X, td.Y = mb.Width(), 2*mb.Height()/3
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textKorean, "UnifontMedium", p.Fm.EnsureKey("UnifontMedium")
	td.X, td.Y = 0, mb.Height()/3
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontName, td.FontKey = textVietnamese, "Roboto-Regular", p.Fm.EnsureKey("Roboto-Regular")
	td.X, td.Y = mb.Width(), 0
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 3*mb.Width()/4)

	td.Text, td.FontSize, td.ShowTextBB = textEnglish, 24, false
	td.X, td.Y, td.HAlign = -1, -1, pdf.AlignCenter
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 0)

	td.FontSize = 80
	td.Text, td.HAlign, td.X, td.Y = "C", pdf.AlignRight, mb.Width(), mb.Height()
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "J", pdf.AlignLeft, 0, 2*mb.Height()/3
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "K", pdf.AlignRight, mb.Width(), mb.Height()/3
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 0)

	td.Text, td.HAlign, td.X, td.Y = "V", pdf.AlignLeft, 0, 0
	pdf.WriteColumn(p.Buf, mediaBox, nil, td, 0)

	return p
}

func TestCJKV(t *testing.T) {
	msg := "TestCJKV"

	// Install test user fonts (in addition to already installed user fonts)
	// from pkg/testdata/fonts.
	api.LoadConfiguration()
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	mediaBox := pdf.RectForDim(600, 600)
	createXRefAndWritePDF(t, msg, "UserFont_CJKV", createCJKVDemo(mediaBox))
}
