/*
Copyright 2018 The pdfcpu Authors.

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

package pdfcpu

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
)

// ICC profiles are not yet supported!
//
// We fall back to the alternate color space and if there is none to whatever color space makes sense.

//ICC profiles use big endian always.
type iccProfile struct {
	b          []byte
	rX, rY, rZ float32 // redMatrixColumn; the first column in the matrix, which is used in matrix/TRC transforms.
	gX, gY, gZ float32 // greenMatrixColumn; the second column in the matrix, which is used in matrix/TRC transforms.
	bX, bY, bZ float32 // blueMatrixColumn; the third column in the matrix, which is used in matrix/TRC transforms.
	//TRC = tone reproduction curve
}

// header 128 bytes
// tagcount 4 bytes
// tagtable signature4, offset4, size4(%4=0)
// elements (required, optional, private)

// dateTimeNumber 12 Bytes
// positionNumber offset 4 Bytes size 4 bytes
// response16Number
// s15Fixed16Number

// elementdata 4byte boundary padding

// required:
// profileDescriptionTag
// copyrightTag
// chromaticAdaptationTag

// BToA0Tag ***
// AToB0Tag

func (p iccProfile) tag(sig string) (int, int, error) {

	for i, j := 0, 132; i < p.tagCount(); i++ {
		s := string(p.b[j : j+4])
		if s != sig {
			j += 12
			continue
		}
		j += 4
		off := binary.BigEndian.Uint32(p.b[j:])
		j += 4
		size := binary.BigEndian.Uint32(p.b[j:])
		return int(off), int(size), nil
	}

	return 0, 0, errors.Errorf("tag %s not found", sig)
}

func (p *iccProfile) matrixCol(sig string) (float32, float32, float32, error) {

	off, size, err := p.tag(sig)
	if err != nil {
		return 0, 0, 0, err
	}

	if size != 20 {
		return 0, 0, 0, errors.Errorf("tag %s should have size 20, has:%d", sig, size)
	}

	x, y, z := p.xyz(off + 8)

	return x, y, z, nil
}

func (p *iccProfile) init() error {

	var err error

	p.rX, p.rY, p.rZ, err = p.matrixCol("rXYZ")
	if err != nil {
		return err
	}

	p.gX, p.gY, p.gZ, err = p.matrixCol("gXYZ")
	if err != nil {
		return err
	}

	p.bX, p.bY, p.bZ, err = p.matrixCol("bXYZ")

	return err
}

func (p iccProfile) size() uint32 {
	return binary.BigEndian.Uint32(p.b[0:])
}

func (p iccProfile) preferredCMM() string {
	return string(p.b[4:8])
}

func (p iccProfile) version() string {
	major := p.b[8]
	minor := p.b[9] >> 4
	bugfix := p.b[9] & 0x0F
	return fmt.Sprintf("%d.%d.%d.0", major, minor, bugfix)
}

func (p iccProfile) class() string {
	return string(p.b[12:16])
}

func (p iccProfile) dataColorSpace() string {
	return string(p.b[16:20])
}

func (p iccProfile) pcs() string {
	return string(p.b[20:24])
}

func (p iccProfile) creationTS() string {

	y := binary.BigEndian.Uint16(p.b[24:])
	m := binary.BigEndian.Uint16(p.b[26:])
	d := binary.BigEndian.Uint16(p.b[28:])
	h := binary.BigEndian.Uint16(p.b[30:])
	min := binary.BigEndian.Uint16(p.b[32:])
	s := binary.BigEndian.Uint16(p.b[34:])

	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d", y, m, d, h, min, s)
}

func (p iccProfile) fileSig() string {
	return string(p.b[36:40])
}

func (p iccProfile) primaryPlatform() string {
	return string(p.b[40:44])
}

func (p iccProfile) deviceManufacturer() string {
	return string(p.b[48:52])
}

func (p iccProfile) deviceModel() string {
	return string(p.b[52:56])
}

func (p iccProfile) renderingIntent() string {
	ri := binary.BigEndian.Uint16(p.b[66:])
	switch ri {
	case 0:
		return "Perceptual"
	case 1:
		return "Media-relative colorimetric"
	case 2:
		return "Saturation"
	case 3:
		return "ICC-absolute colorimetric"

	}
	return "Perceptual"
}

func (p iccProfile) xyz(i int) (x, y, z float32) {

	x = float32(binary.BigEndian.Uint16(p.b[i:]))
	f := float32(binary.BigEndian.Uint16(p.b[i+2:])) / 0x10000
	if x < 0 {
		x -= f
	} else {
		x += f
	}
	i += 4

	y = float32(binary.BigEndian.Uint16(p.b[i:]))
	f = float32(binary.BigEndian.Uint16(p.b[i+2:])) / 0x10000
	if y < 0 {
		y -= f
	} else {
		y += f
	}
	i += 4

	z = float32(binary.BigEndian.Uint16(p.b[i:]))
	f = float32(binary.BigEndian.Uint16(p.b[i+2:])) / 0x10000
	if z < 0 {
		z -= f
	} else {
		z += f
	}

	return
}

func (p iccProfile) PCSIlluminant() string {

	x, y, z := p.xyz(68)

	return fmt.Sprintf("X=%4.4f Y=%4.4f Z=%4.4f", x, y, z)
}

func (p iccProfile) creator() string {
	return string(p.b[80:84])
}

func (p iccProfile) id() string {
	return hex.EncodeToString(p.b[84:100])
}

func (p iccProfile) tagCount() int {
	return int(binary.BigEndian.Uint32(p.b[128:]))
}

func (p iccProfile) String() string {

	// profile size: 4 bytes at offset 0 (uintt32)
	s := fmt.Sprintf(""+
		"              size: %d\n"+
		"      preferredCMM: %s\n"+
		"           version: %s\n"+
		"             class: %s\n"+
		"            dataCS: %s\n"+
		"               pcs: %s\n"+
		"        creationTS: %s\n"+
		"           fileSig: %s\n"+
		"      primPlatform: %s\n"+
		"deviceManufacturer: %s\n"+
		"       deviceModel: %s\n"+
		"  rendering intent: %s\n"+
		"    PCS illuminant: %s\n"+
		"           creator: %s\n"+
		"                id: %s\n"+
		"          tagCount: %d\n\n",
		p.size(),
		p.preferredCMM(),
		p.version(),
		p.class(),
		p.dataColorSpace(),
		p.pcs(),
		p.creationTS(),
		p.fileSig(),
		p.primaryPlatform(),
		p.deviceManufacturer(),
		p.deviceModel(),
		p.renderingIntent(),
		p.PCSIlluminant(),
		p.creator(),
		p.id(),
		p.tagCount(),
	)

	for i, j := 0, 132; i < p.tagCount(); i++ {
		sig := string(p.b[j : j+4])
		j += 4
		off := binary.BigEndian.Uint32(p.b[j:])
		j += 4
		size := binary.BigEndian.Uint32(p.b[j:])
		j += 4
		s += fmt.Sprintf("Tag %d: signature:%s offset:%d(#%02x) size:%d(#%02x)\n%s\n", i, sig, off, off, size, size, hex.Dump(p.b[off:off+size]))
		//s += fmt.Sprintf("Tag %d: signature:%s offset:%d(#%02x) size:%d(#%02x)\n", i, sig, off, off, size, size)
	}
	s += fmt.Sprintf("Matrix:\n")
	s += fmt.Sprintf("%4.4f %4.4f %4.4f\n", p.rX, p.gX, p.bX)
	s += fmt.Sprintf("%4.4f %4.4f %4.4f\n", p.rY, p.gY, p.bY)
	s += fmt.Sprintf("%4.4f %4.4f %4.4f\n", p.rZ, p.gZ, p.bZ)

	// cprt copyrightTag multiLocalizedUnicodeType contains the text copyright information for the profile.
	// desc profileDescriptionTag multiLocalizedUnicodeType describes the structure containing invariant and localizable versions of the profile description for display. => 10.13

	// wtpt mediaWhitePointTag XYZType used for generating the ICC-absolute colorimetric intent, specifies the chromatically adapted nCIEXYZ tristimulus values of the media white point.
	// bkpt

	// rXYZ XYZType redMatrixColumnTag  	contains the first column in the matrix used in matrix/TRC transforms.
	// gXYZ XYZType greenMatrixColumnTag  	contains the second column in the matrix used in matrix/TRC transforms.
	// bXYZ XYZType blueMatrixColumnTag   	contains the third column in the matrix used in matrix/TRC transforms.

	// rTRC curveType or parametricCurveType redTRCTag				contains the red channel tone reproduction curve. f(device)=linear
	// gTRC curveType or parametricCurveType greenTRCTag			contains the green channel tone reproduction curve.
	// bTRC curveType or parametricCurveType blueTRCTag				contains the blue channel tone reproduction curve.

	// dmnd deviceMfgDescTag	multiLocalizedUnicodeType	describes the structure containing invariant and localizable versions of the device manufacturer for display. => 10.13
	// dmdd deviceModelDescTag	multiLocalizedUnicodeType	describes the structure containing invariant and localizable versions of the device model for display. => 10.13
	// vued viewingCondDescTag		describes the structure containing invariant and localizable versions of the viewing conditions. => 10.13

	// view viewingConditionsTag viewingConditionsType defines the viewing conditions parameters. => 10.28
	// lumi luminanceTag XYZType contains the absolute luminance of emissive devices in candelas per square metre as described by the Y channel.
	// meas measurementTag measurementType describes the alternative measurement specification, such as a D65 illuminant instead of the default D50.
	// tech technologyTag signatureType => table 29

	return s
}
