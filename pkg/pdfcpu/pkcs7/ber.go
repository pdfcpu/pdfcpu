/*
Copyright (c) 2015 Andrew Smith
Copyright 2026 The pdfcpu Authors.

Licensed under the MIT License. See LICENSE in this directory.
*/

package pkcs7

import (
	"bytes"
	"errors"
	"fmt"
)

type asn1Object interface {
	EncodeTo(writer *bytes.Buffer) error
}

type asn1Structured struct {
	tagBytes []byte
	content  []asn1Object
}

func (s asn1Structured) EncodeTo(out *bytes.Buffer) error {
	inner := new(bytes.Buffer)
	for i, obj := range s.content {
		if obj == nil {
			return fmt.Errorf("encode child %d: missing ASN.1 object", i+1)
		}
		if err := obj.EncodeTo(inner); err != nil {
			return fmt.Errorf("encode child %d: %w", i+1, err)
		}
	}
	if _, err := out.Write(s.tagBytes); err != nil {
		return fmt.Errorf("write tag: %w", err)
	}
	if err := encodeLength(out, inner.Len()); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := out.Write(inner.Bytes()); err != nil {
		return fmt.Errorf("write content: %w", err)
	}
	return nil
}

type asn1Primitive struct {
	tagBytes []byte
	content  []byte
}

func (p asn1Primitive) EncodeTo(out *bytes.Buffer) error {
	_, err := out.Write(p.tagBytes)
	if err != nil {
		return fmt.Errorf("write tag: %w", err)
	}
	if err = encodeLength(out, len(p.content)); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err = out.Write(p.content); err != nil {
		return fmt.Errorf("write content: %w", err)
	}
	return nil
}

func ber2der(ber []byte) ([]byte, error) {
	if len(ber) == 0 {
		return nil, errors.New("ber2der: input ber is empty")
	}
	out := new(bytes.Buffer)

	obj, next, err := readObject(ber, 0)
	if err != nil {
		return nil, fmt.Errorf("ber2der: read object at offset 0: %w", err)
	}
	if next != len(ber) && !allZero(ber[next:]) {
		return nil, fmt.Errorf("ber2der: trailing data at offset %d", next)
	}
	if err := obj.EncodeTo(out); err != nil {
		return nil, fmt.Errorf("ber2der: encode DER: %w", err)
	}

	return out.Bytes(), nil
}

func allZero(bb []byte) bool {
	for _, b := range bb {
		if b != 0 {
			return false
		}
	}
	return true
}

// encodes lengths that are longer than 127 into string of bytes
func marshalLongLength(out *bytes.Buffer, i int) (err error) {
	n := lengthLength(i)

	for ; n > 0; n-- {
		err = out.WriteByte(byte(i >> uint((n-1)*8)))
		if err != nil {
			return
		}
	}

	return nil
}

// computes the byte length of an encoded length value
func lengthLength(i int) (numBytes int) {
	numBytes = 1
	for i > 255 {
		numBytes++
		i >>= 8
	}
	return
}

// encodes the length in DER format
// If the length fits in 7 bits, the value is encoded directly.
//
// Otherwise, the number of bytes to encode the length is first determined.
// This number is likely to be 4 or less for a 32bit length. This number is
// added to 0x80. The length is encoded in big endian encoding follow after
//
// Examples:
//
//	length | byte 1 | bytes n
//	0      | 0x00   | -
//	120    | 0x78   | -
//	200    | 0x81   | 0xC8
//	500    | 0x82   | 0x01 0xF4
func encodeLength(out *bytes.Buffer, length int) (err error) {
	if length < 0 {
		return errors.New("negative length")
	}
	if length >= 128 {
		l := lengthLength(length)
		err = out.WriteByte(0x80 | byte(l))
		if err != nil {
			return
		}
		err = marshalLongLength(out, length)
		if err != nil {
			return
		}
	} else {
		err = out.WriteByte(byte(length))
		if err != nil {
			return
		}
	}
	return
}

func readObject(ber []byte, offset int) (asn1Object, int, error) {
	start := offset
	tagStart, tagEnd, constructed, next, err := readTag(ber, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("read tag at offset %d: %w", start, err)
	}
	length, indefinite, offset, err := readLength(ber, next)
	if err != nil {
		return nil, 0, fmt.Errorf("read length at offset %d: %w", next, err)
	}
	if tagEnd-tagStart == 1 && ber[tagStart] == 0 && !indefinite && length == 0 {
		return nil, 0, errors.New("unexpected end-of-content marker")
	}
	if length > len(ber)-offset {
		return nil, 0, errors.New("content length exceeds available data")
	}
	contentEnd := offset + length
	if indefinite && !constructed {
		return nil, 0, errors.New("indefinite length requires constructed encoding")
	}

	if !constructed {
		return asn1Primitive{
			tagBytes: ber[tagStart:tagEnd],
			content:  ber[offset:contentEnd],
		}, contentEnd, nil
	}

	content, contentEnd, err := readStructuredContent(ber, offset, contentEnd, indefinite)
	if err != nil {
		return nil, 0, fmt.Errorf("read constructed content at offset %d: %w", offset, err)
	}
	return asn1Structured{tagBytes: ber[tagStart:tagEnd], content: content}, contentEnd, nil
}

func readTag(ber []byte, offset int) (start, end int, constructed bool, next int, err error) {
	if offset < 0 || offset >= len(ber) {
		return 0, 0, false, 0, errors.New("offset outside BER data")
	}
	start = offset
	first := ber[offset]
	offset++
	if first&0x1f != 0x1f {
		return start, offset, first&0x20 != 0, offset, nil
	}
	firstOctet := true
	for {
		if offset >= len(ber) {
			return 0, 0, false, 0, errors.New("unterminated high-tag-number")
		}
		current := ber[offset]
		offset++
		if firstOctet && current == 0x80 {
			return 0, 0, false, 0, errors.New("high-tag-number has leading zero")
		}
		firstOctet = false
		if current < 0x80 {
			return start, offset, first&0x20 != 0, offset, nil
		}
	}
}

func readLength(ber []byte, offset int) (length int, indefinite bool, next int, err error) {
	if offset < 0 || offset >= len(ber) {
		return 0, false, 0, errors.New("length offset outside BER data")
	}
	first := ber[offset]
	offset++
	if first == 0x80 {
		return 0, true, offset, nil
	}
	if first < 0x80 {
		return int(first), false, offset, nil
	}
	count := int(first & 0x7f)
	if count > 4 {
		return 0, false, 0, errors.New("length uses more than four octets")
	}
	if count > len(ber)-offset {
		return 0, false, 0, errors.New("length octets exceed available data")
	}
	if ber[offset] == 0 {
		return 0, false, 0, errors.New("length has leading zero")
	}
	if count == 4 && ber[offset] > 0x7f {
		return 0, false, 0, errors.New("length exceeds supported range")
	}
	for _, b := range ber[offset : offset+count] {
		length = length*256 + int(b)
	}
	return length, false, offset + count, nil
}

func readStructuredContent(
	ber []byte,
	offset, contentEnd int,
	indefinite bool,
) ([]asn1Object, int, error) {
	if offset < 0 || contentEnd < offset || contentEnd > len(ber) {
		return nil, 0, errors.New("constructed-content boundary outside BER data")
	}
	var content []asn1Object
	for indefinite || offset < contentEnd {
		if indefinite {
			terminated, err := isIndefiniteTermination(ber, offset)
			if err != nil {
				return nil, 0, err
			}
			if terminated {
				return content, offset + 2, nil
			}
		}
		obj, next, err := readObject(ber, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("read child at offset %d: %w", offset, err)
		}
		if next <= offset {
			return nil, 0, fmt.Errorf("child at offset %d did not advance", offset)
		}
		if !indefinite && next > contentEnd {
			return nil, 0, fmt.Errorf("child at offset %d exceeds parent boundary %d", offset, contentEnd)
		}
		content = append(content, obj)
		offset = next
	}
	return content, contentEnd, nil
}

func isIndefiniteTermination(ber []byte, offset int) (bool, error) {
	if offset < 0 || offset > len(ber) || len(ber)-offset < 2 {
		return false, errors.New("missing indefinite-length terminator")
	}

	return ber[offset] == 0 && ber[offset+1] == 0, nil
}
