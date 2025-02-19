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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var inDir, outDir string
var xRefTable *model.XRefTable

func TestMain(m *testing.M) {

	inDir = filepath.Join("..", "testdata", "resources")

	var err error

	xRefTable, err = CreateXRefTableWithRootDict()
	if err != nil {
		os.Exit(1)
	}

	outDir, err = os.MkdirTemp("", "pdfcpu_imageTests")
	if err != nil {
		os.Exit(1)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

func streamDictForJPGFile(xRefTable *model.XRefTable, fileName string) (*types.StreamDict, error) {

	bb, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	c, _, err := image.DecodeConfig(bytes.NewReader(bb))
	if err != nil {
		return nil, err
	}

	var cs string

	switch c.ColorModel {

	case color.GrayModel:
		cs = model.DeviceGrayCS

	case color.YCbCrModel:
		cs = model.DeviceRGBCS

	case color.CMYKModel:
		cs = model.DeviceCMYKCS

	default:
		return nil, errors.New("pdfcpu: unexpected color model for JPEG")

	}

	sd, err := model.CreateDCTImageStreamDict(xRefTable, bb, c.Width, c.Height, 8, cs)
	if err != nil {
		return nil, err
	}

	// Ensure decoded image stream.
	if err := sd.Decode(); err != nil {
		return nil, err
	}

	return sd, nil
}

func streamDictForImageFile(xRefTable *model.XRefTable, fileName string) (*types.StreamDict, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sd, _, _, err := model.CreateImageStreamDict(xRefTable, f)
	return sd, err
}

func compare(t *testing.T, fn1, fn2 string) {

	f1, err := os.Open(fn1)
	if err != nil {
		t.Errorf("%s: %v", fn1, err)
		return
	}
	defer f1.Close()

	bb1, err := io.ReadAll(f1)
	if err != nil {
		t.Errorf("%s: %v", fn1, err)
		return
	}

	f2, err := os.Open(fn2)
	if err != nil {
		t.Errorf("%s: %v", fn2, err)
		return
	}
	defer f1.Close()

	bb2, err := io.ReadAll(f2)
	if err != nil {
		t.Errorf("%s: %v", fn2, err)
		return
	}

	if len(bb1) != len(bb2) {
		t.Errorf("%s <-> %s: length mismatch %d != %d", fn1, fn2, len(bb1), len(bb2))
		return
	}

	for i := 0; i < len(bb1); i++ {
		if bb1[i] != bb2[i] {
			t.Errorf("%s <-> %s: mismatch at %d, 0x%02x != 0x%02x\n", fn1, fn2, i, bb1[i], bb2[i])
			return
		}
	}

}

func printOptionalSMask(t *testing.T, sd *types.StreamDict) {
	o := sd.IndirectRefEntry("SMask")
	if o != nil {
		sm, err := xRefTable.Dereference(*o)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}
		fmt.Printf("SMask %s: %s\n", o, sm)
	}
}
func TestReadWritePNGAndWEBP(t *testing.T) {

	for _, filename := range []string{
		"mountain.png",
		"mountain.webp",
	} {

		// Read a PNG file and create an image object which is a stream dict.
		sd, err := streamDictForImageFile(xRefTable, filepath.Join(inDir, filename))
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Print the image object.
		fmt.Printf("created imageObj: %s\n", sd)

		// Print the optional SMask.
		printOptionalSMask(t, sd)

		// The file type and its extension gets decided during the call to WriteImage!
		// These testcases all produce PNG files.
		fnNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		tmpFileName1 := filepath.Join(outDir, fnNoExt)
		fmt.Printf("tmpFileName: %s\n", tmpFileName1)

		// Write the image object (as PNG file) to disk.
		// fn1 is the resulting fileName path including the suffix (aka filetype extension).
		fn1, err := WriteImage(xRefTable, tmpFileName1, sd, false, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
		// we can only compare against a PNG file which resulted from using image/png.

		// Read in a PNG file created by pdfcpu and create an image object.
		sd, err = streamDictForImageFile(xRefTable, fn1)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Write the image object (as PNG file) to disk.s
		fn2, err := WriteImage(xRefTable, tmpFileName1+"2", sd, false, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// ..and compare each other.
		compare(t, fn1, fn2)
	}

}

// Read in a device gray image stream dump from disk.
func read1BPCDeviceGrayFlateStreamDump(xRefTable *model.XRefTable, fileName string) (*types.StreamDict, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read in a flate encoded stream.
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("XObject"),
				"Subtype":          types.Name("Image"),
				"Width":            types.Integer(1161),
				"Height":           types.Integer(392),
				"BitsPerComponent": types.Integer(1),
				"ColorSpace":       types.Name(model.DeviceGrayCS),
				"Decode":           types.NewNumberArray(1, 0),
			},
		),
		Raw:            buf,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}}}

	sd.InsertName("Filter", filter.Flate)

	return sd, sd.Decode()
}

// Starting out with a DeviceGray color space based image object, write a PNG file then read and write again.
func TestReadDeviceGrayWritePNG(t *testing.T) {

	// Create an image for a flate encoded stream dump file.
	filename := "DeviceGray"
	path := filepath.Join(inDir, filename+".raw")

	sd, err := read1BPCDeviceGrayFlateStreamDump(xRefTable, path)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Print the image object.
	fmt.Printf("created imageObj: %s\n", sd)
	o := sd.IndirectRefEntry("SMask")
	if o != nil {
		sm, err := xRefTable.Dereference(*o)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}
		fmt.Printf("SMask %s: %s\n", o, sm)
	}

	tmpFile1 := filepath.Join(outDir, filename)

	// Write the image object as PNG file.
	fn1, err := WriteImage(xRefTable, tmpFile1, sd, false, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
	// we can only compare against a PNG file which resulted from using image/png.

	// Read in a PNG file created by pdfcpu and create an image object.
	sd, err = streamDictForImageFile(xRefTable, fn1)
	if err != nil || sd == nil {
		t.Fatalf("err: %v\n", err)
	}

	fmt.Printf("created another imageObj: %s\n", sd)

	tmpFile2 := filepath.Join(outDir, filename+"2")

	// Write the image object as PNG file.
	fn2, err := WriteImage(xRefTable, tmpFile2, sd, false, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// ..and compare each other.
	compare(t, fn1, fn2)
}

// Read in a device CMYK image stream dump from disk.
func read8BPCDeviceCMYKFlateStreamDump(xRefTable *model.XRefTable, fileName string) (*types.StreamDict, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	decodeParms := types.Dict(
		map[string]types.Object{
			"BitsPerComponent": types.Integer(8),
			"Colors":           types.Integer(4),
			"Columns":          types.Integer(340),
		},
	)

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("XObject"),
				"Subtype":          types.Name("Image"),
				"Width":            types.Integer(340),
				"Height":           types.Integer(216),
				"BitsPerComponent": types.Integer(8),
				"ColorSpace":       types.Name(model.DeviceCMYKCS),
			},
		),
		Raw:            buf,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: decodeParms}}}

	sd.InsertName("Filter", filter.Flate)

	sd.FilterPipeline[0].DecodeParms = decodeParms

	return sd, sd.Decode()
}

// Starting out with a CMYK color space based image object, write a TIFF file then read and write again.
func TestReadCMYKWriteTIFF(t *testing.T) {

	filename := "DeviceCMYK"
	path := filepath.Join(inDir, filename+".raw")

	sd, err := read8BPCDeviceCMYKFlateStreamDump(xRefTable, path)
	if err != nil {
		t.Errorf("err: %v\n", err)
	}

	// Print the image object.
	fmt.Printf("created imageObj: %s\n", sd)

	// Print the optional SMask.
	printOptionalSMask(t, sd)

	// The file type and its extension gets decided during WriteImage.
	// These testcases all produce TIFF files.
	tmpFile1 := filepath.Join(outDir, filename)

	// Write the image object as TIFF file.
	fn1, err := WriteImage(xRefTable, tmpFile1, sd, false, 0)
	if err != nil {
		t.Errorf("err: %v\n", err)
	}

	// Read in a TIFF file created by pdfcpu and create an image object.
	sd, err = streamDictForImageFile(xRefTable, fn1)
	if err != nil || sd == nil {
		t.Errorf("err: %v\n", err)
	}

	tmpFile2 := filepath.Join(outDir, filename+"2")

	// Write the image object as TIFF file.
	fn2, err := WriteImage(xRefTable, tmpFile2, sd, false, 0)
	if err != nil {
		t.Errorf("err: %v\n", err)
	}

	// ..and compare each other.
	compare(t, fn1, fn2)

}

func TestReadTIFFWritePNG(t *testing.T) {

	// TIFF images get read into a Flate encoded image stream like PNGs.
	// Any Flate encoded image stream gets written as PNG unless it operates in the Device CMYK color space.

	fileName := "mountain.tif"

	// Read a TIFF file and create an image object which is a stream dict.
	sd, err := streamDictForImageFile(xRefTable, filepath.Join(inDir, fileName))
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Print the image object.
	fmt.Printf("created imageObj: %s\n", sd)

	// Print the optional SMask.
	printOptionalSMask(t, sd)

	// The file type and its extension gets decided during the call to WriteImage!
	// These testcases all produce PNG files.
	fnNoExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	tmpFileName1 := filepath.Join(outDir, fnNoExt)
	fmt.Printf("tmpFileName: %s\n", tmpFileName1)

	// Write the image object (as PNG file) to disk.
	// fn1 is the resulting fileName path including the suffix (aka filetype extension).
	fn1, err := WriteImage(xRefTable, tmpFileName1, sd, false, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
	// we can only compare against a PNG file which resulted from using image/png.

	// Read in a PNG file created by pdfcpu and create an image object.
	sd, err = streamDictForImageFile(xRefTable, fn1)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Write the image object (as PNG file) to disk.
	fn2, err := WriteImage(xRefTable, tmpFileName1+"2", sd, false, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// ..and compare each other.
	compare(t, fn1, fn2)
}

func TestReadWriteJPEG(t *testing.T) {

	fileName := "mountain.jpg"

	// Read a JPEG file and create a stream dict w/o decoding.
	sd, err := streamDictForJPGFile(xRefTable, filepath.Join(inDir, fileName))
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Print the image object.
	fmt.Printf("created imageObj: %s\n", sd)

	// Print the optional SMask.
	printOptionalSMask(t, sd)

	// The file type and its extension gets decided during the call to WriteImage!
	// These testcases all produce PNG files.
	fnNoExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	tmpFileName1 := filepath.Join(outDir, fnNoExt)
	fmt.Printf("tmpFileName: %s\n", tmpFileName1)

	// Write the image object (as .jpg file) to disk.
	// fn is the resulting fileName path including the suffix (aka filetype extension).
	fn, err := WriteImage(xRefTable, tmpFileName1, sd, false, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	fmt.Printf("fileName: %s\n", fn)
	// No comparison since JPG is lossy.
}
