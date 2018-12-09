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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hhrutter/pdfcpu/pkg/filter"
)

var inDir, outDir string
var xRefTable *XRefTable

func TestMain(m *testing.M) {

	inDir = "testdata"

	var err error

	xRefTable, err = createXRefTableWithRootDict()
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	outDir, err = ioutil.TempDir("", "pdfcpu_imageTests")
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

func compare(t *testing.T, fn1, fn2 string) {

	f1, err := os.Open(fn1)
	if err != nil {
		t.Errorf("%s: %v", fn1, err)
		return
	}
	defer f1.Close()

	bb1, err := ioutil.ReadAll(f1)
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

	bb2, err := ioutil.ReadAll(f2)
	if err != nil {
		t.Errorf("%s: %v", fn2, err)
		return
	}

	if len(bb1) != len(bb1) {
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

func printOptionalSMask(t *testing.T, sd *StreamDict) {
	o := sd.IndirectRefEntry("SMask")
	if o != nil {
		sm, err := xRefTable.Dereference(*o)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}
		fmt.Printf("SMask %s: %s\n", o, sm)
	}
}
func TestReadWritePNG(t *testing.T) {

	for _, filename := range []string{
		"demo.png",     // fully opaque
		"pdfchip3.png", // transparent
		"DeviceGray.png",
	} {

		// Read a PNG file and create an image object which is a stream dict.
		sd, err := ReadPNGFile(xRefTable, filepath.Join(inDir, filename))
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
		fn1, err := WriteImage(xRefTable, tmpFileName1, sd, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
		// we can only compare against a PNG file which resulted from using image/png.

		// Read in a PNG file created by pdfcpu and create an image object.
		sd, err = ReadPNGFile(xRefTable, fn1)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Write the image object (as PNG file) to disk.s
		fn2, err := WriteImage(xRefTable, tmpFileName1+"2", sd, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// ..and compare each other.
		compare(t, fn1, fn2)
	}

}

// Read in a device gray image stream dump from disk.
func read1BPCDeviceGrayFlateStreamDump(xRefTable *XRefTable, fileName string) (*StreamDict, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read in a flate encoded stream.
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"Width":            Integer(1161),
				"Height":           Integer(392),
				"BitsPerComponent": Integer(1),
				"ColorSpace":       Name(DeviceGrayCS),
				"Decode":           NewNumberArray(1, 0),
			},
		),
		Raw:            buf,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}}

	sd.InsertName("Filter", filter.Flate)

	err = decodeStream(sd)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

// Starting out with a DeviceGray color space based image object, write a PNG file then read and write again.
func TestReadImageStreamWritePNG(t *testing.T) {

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
	fn1, err := WriteImage(xRefTable, tmpFile1, sd, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
	// we can only compare against a PNG file which resulted from using image/png.

	// Read in a PNG file created by pdfcpu and create an image object.
	sd, err = ReadPNGFile(xRefTable, fn1)
	if err != nil || sd == nil {
		t.Fatalf("err: %v\n", err)
	}

	fmt.Printf("created another imageObj: %s\n", sd)

	tmpFile2 := filepath.Join(outDir, filename+"2")

	// Write the image object as PNG file.
	fn2, err := WriteImage(xRefTable, tmpFile2, sd, 0)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	// ..and compare each other.
	compare(t, fn1, fn2)
}

// Read in a device CMYK image stream dump from disk.
func read8BPCDeviceCMYKFlateStreamDump(xRefTable *XRefTable, fileName string) (*StreamDict, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	decodeParms := Dict(
		map[string]Object{
			"BitsPerComponent": Integer(8),
			"Colors":           Integer(4),
			"Columns":          Integer(340),
		},
	)

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"Width":            Integer(340),
				"Height":           Integer(216),
				"BitsPerComponent": Integer(8),
				"ColorSpace":       Name(DeviceCMYKCS),
			},
		),
		Raw:            buf,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: decodeParms}}}

	sd.InsertName("Filter", filter.Flate)

	err = decodeStream(sd)
	if err != nil {
		return nil, err
	}

	return sd, nil
}

// Starting out with a CMYK color space based image object, write a TIFF file then read and write again.
func TestReadImageStreamWriteTIFF(t *testing.T) {

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
	fn1, err := WriteImage(xRefTable, tmpFile1, sd, 0)
	if err != nil {
		t.Errorf("err: %v\n", err)
	}

	// Read in a TIFF file created by pdfcpu and create an image object.
	sd, err = ReadTIFFFile(xRefTable, fn1)
	if err != nil || sd == nil {
		t.Errorf("err: %v\n", err)
	}

	tmpFile2 := filepath.Join(outDir, filename+"2")

	// Write the image object as TIFF file.
	fn2, err := WriteImage(xRefTable, tmpFile2, sd, 0)
	if err != nil {
		t.Errorf("err: %v\n", err)
	}

	// ..and compare each other.
	compare(t, fn1, fn2)

}

func TestReadTIFFWritePNG(t *testing.T) {

	// TIFF images get read into a Flate encoded image stream like PNGs.
	// Any Flate encoded image stream gets written as PNG unless it operates in the Device CMYK color space.

	for _, filename := range []string{
		"video-001.tiff",
		"24bit_1800dpi.tif",
		"48bit_1800dpi.tif",
	} {

		// Read a TIFF file and create an image object which is a stream dict.
		sd, err := ReadTIFFFile(xRefTable, filepath.Join(inDir, filename))
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
		fn1, err := WriteImage(xRefTable, tmpFileName1, sd, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Since image/png does not write all ancillary chunks (eg. pHYs for dpi)
		// we can only compare against a PNG file which resulted from using image/png.

		// Read in a PNG file created by pdfcpu and create an image object.
		sd, err = ReadPNGFile(xRefTable, fn1)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// Write the image object (as PNG file) to disk.
		fn2, err := WriteImage(xRefTable, tmpFileName1+"2", sd, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		// ..and compare each other.
		compare(t, fn1, fn2)
	}

}

func TestReadWriteJPEG(t *testing.T) {

	for _, filename := range []string{
		"video-001.221212.jpeg",
		"video-001.cmyk.jpeg",
		"video-001.jpeg",
		"video-001.progressive.jpeg",
		"video-005.gray.jpeg",
	} {

		// Read a JPEG file and create an image object which is a stream dict.
		sd, err := ReadJPEGFile(xRefTable, filepath.Join(inDir, filename))
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

		// Write the image object (as .jpg file) to disk.
		// fn is the resulting fileName path including the suffix (aka filetype extension).
		fn, err := WriteImage(xRefTable, tmpFileName1, sd, 0)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}
		fmt.Printf("fileName: %s\n", fn)
	}
}
