package filter

import (
	"bytes"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math/rand"
	"testing"
)

func randJPEG() ([]byte, error) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for i := range img.Pix {
		img.Pix[i] = uint8(rand.Int())
	}
	err := jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), err
}

func TestDCT(t *testing.T) {
	for range [10]int{} {
		b, err := randJPEG()
		if err != nil {
			t.Fatal(err)
		}

		additional := make([]byte, 250)
		rand.Read(additional)

		b = append(b, additional...)

		rd := bytes.NewReader(b)
		lim := LimitedDCTDecoder(rd)

		_, err = jpeg.Decode(lim)
		if err != nil {
			t.Fatal(err)
		}

		if rd.Len() != len(additional) {
			t.Errorf("expected %d got %d", rd.Len(), len(additional))
		}
	}
}

func TestDCTFail(t *testing.T) {
	for range [30]int{} {
		input := make([]byte, 25000)
		rand.Read(input)

		lim := LimitedDCTDecoder(bytes.NewReader(input))
		_, err := ioutil.ReadAll(lim)
		if err == nil {
			t.Error("expected error on random input data")
		}
	}
}
