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

// Functions dealing with PDF encryption.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

var (
	pad = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}

	nullPad32 = make([]byte, 32)

	// Needed permission bits for pdfcpu commands.
	perm = map[CommandMode]struct{ extract, modify int }{
		VALIDATE:                {0, 0},
		OPTIMIZE:                {0, 0},
		SPLIT:                   {1, 0},
		MERGECREATE:             {0, 0},
		MERGEAPPEND:             {0, 0},
		EXTRACTIMAGES:           {1, 0},
		EXTRACTFONTS:            {1, 0},
		EXTRACTPAGES:            {1, 0},
		EXTRACTCONTENT:          {1, 0},
		EXTRACTMETADATA:         {1, 0},
		TRIM:                    {0, 1},
		LISTATTACHMENTS:         {0, 0},
		EXTRACTATTACHMENTS:      {1, 0},
		ADDATTACHMENTS:          {0, 1},
		ADDATTACHMENTSPORTFOLIO: {0, 1},
		REMOVEATTACHMENTS:       {0, 1},
		LISTPERMISSIONS:         {0, 0},
		SETPERMISSIONS:          {0, 0},
		ADDWATERMARKS:           {0, 1},
		REMOVEWATERMARKS:        {0, 1},
		INSERTPAGESBEFORE:       {0, 1},
		INSERTPAGESAFTER:        {0, 1},
		REMOVEPAGES:             {0, 1},
		LISTKEYWORDS:            {0, 0},
		ADDKEYWORDS:             {0, 1},
		REMOVEKEYWORDS:          {0, 1},
		LISTPROPERTIES:          {0, 0},
		ADDPROPERTIES:           {0, 1},
		REMOVEPROPERTIES:        {0, 1},
		COLLECT:                 {1, 0},
		CROP:                    {0, 1},
		LISTBOXES:               {0, 0},
		ADDBOXES:                {0, 1},
		REMOVEBOXES:             {0, 1},
		LISTIMAGES:              {0, 1},
		CREATE:                  {0, 0},
	}
)

// NewEncryptDict creates a new EncryptDict using the standard security handler.
func newEncryptDict(needAES bool, keyLength int, permissions int16) Dict {

	d := NewDict()

	d.Insert("Filter", Name("Standard"))

	if keyLength >= 128 {
		d.Insert("Length", Integer(keyLength))
		i := 4
		if keyLength == 256 {
			i = 5
		}
		d.Insert("R", Integer(i))
		d.Insert("V", Integer(i))
	} else {
		d.Insert("R", Integer(2))
		d.Insert("V", Integer(1))
	}

	// Set user access permission flags.
	d.Insert("P", Integer(permissions))

	d.Insert("StmF", Name("StdCF"))
	d.Insert("StrF", Name("StdCF"))

	d1 := NewDict()
	d1.Insert("AuthEvent", Name("DocOpen"))

	if needAES {
		n := "AESV2"
		if keyLength == 256 {
			n = "AESV3"
		}
		d1.Insert("CFM", Name(n))
	} else {
		d1.Insert("CFM", Name("V2"))
	}

	d1.Insert("Length", Integer(keyLength/8))

	d2 := NewDict()
	d2.Insert("StdCF", d1)

	d.Insert("CF", d2)

	if keyLength == 256 {
		d.Insert("U", NewHexLiteral(make([]byte, 48)))
		d.Insert("O", NewHexLiteral(make([]byte, 48)))
		d.Insert("UE", NewHexLiteral(make([]byte, 32)))
		d.Insert("OE", NewHexLiteral(make([]byte, 32)))
		d.Insert("Perms", NewHexLiteral(make([]byte, 16)))
	} else {
		d.Insert("U", NewHexLiteral(make([]byte, 32)))
		d.Insert("O", NewHexLiteral(make([]byte, 32)))
	}

	return d
}

func encKey(userpw string, e *Enc) (key []byte) {

	// 2a
	pw := []byte(userpw)
	if len(pw) >= 32 {
		pw = pw[:32]
	} else {
		pw = append(pw, pad[:32-len(pw)]...)
	}

	// 2b
	h := md5.New()
	h.Write(pw)

	// 2c
	h.Write(e.O)

	// 2d
	var q = uint32(e.P)
	h.Write([]byte{byte(q), byte(q >> 8), byte(q >> 16), byte(q >> 24)})

	// 2e
	h.Write(e.ID)

	// 2f
	if e.R == 4 && !e.Emd {
		h.Write([]byte{0xff, 0xff, 0xff, 0xff})
	}

	// 2g
	key = h.Sum(nil)

	// 2h
	if e.R >= 3 {
		for i := 0; i < 50; i++ {
			h.Reset()
			h.Write(key[:e.L/8])
			key = h.Sum(nil)
		}
	}

	// 2i
	if e.R >= 3 {
		key = key[:e.L/8]
	} else {
		key = key[:5]
	}

	return key
}

// validateUserPassword validates the user password aka document open password.
func validateUserPassword(ctx *Context) (ok bool, err error) {

	if ctx.E.R == 5 {
		return validateUserPasswordAES256(ctx)
	}

	// Alg.4/5 p63
	// 4a/5a create encryption key using Alg.2 p61

	u, key, err := u(ctx)
	if err != nil {
		return false, err
	}

	ctx.EncKey = key

	switch ctx.E.R {

	case 2:
		ok = bytes.Equal(ctx.E.U, u)

	case 3, 4:
		ok = bytes.HasPrefix(ctx.E.U, u[:16])
	}

	return ok, nil
}

func key(ownerpw, userpw string, r, l int) (key []byte) {

	// 3a
	pw := []byte(ownerpw)
	if len(pw) == 0 {
		pw = []byte(userpw)
	}
	if len(pw) >= 32 {
		pw = pw[:32]
	} else {
		pw = append(pw, pad[:32-len(pw)]...)
	}

	// 3b
	h := md5.New()
	h.Write(pw)
	key = h.Sum(nil)

	// 3c
	if r >= 3 {
		for i := 0; i < 50; i++ {
			h.Reset()
			h.Write(key)
			key = h.Sum(nil)
		}
	}

	// 3d
	if r >= 3 {
		key = key[:l/8]
	} else {
		key = key[:5]
	}

	return key
}

// O calculates the owner password digest.
func o(ctx *Context) ([]byte, error) {

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	e := ctx.E

	// 3a-d
	key := key(ownerpw, userpw, e.R, e.L)

	// 3e
	o := []byte(userpw)
	if len(o) >= 32 {
		o = o[:32]
	} else {
		o = append(o, pad[:32-len(o)]...)
	}

	// 3f
	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	c.XORKeyStream(o, o)

	// 3g
	if e.R >= 3 {
		for i := 1; i <= 19; i++ {
			keynew := make([]byte, len(key))
			copy(keynew, key)

			for j := range keynew {
				keynew[j] ^= byte(i)
			}

			c, err := rc4.NewCipher(keynew)
			if err != nil {
				return nil, err
			}
			c.XORKeyStream(o, o)
		}
	}

	return o, nil
}

// U calculates the user password digest.
func u(ctx *Context) (u []byte, key []byte, err error) {

	// The PW string is generated from OS codepage characters by first converting the string to
	// PDFDocEncoding. If input is Unicode, first convert to a codepage encoding , and then to
	// PDFDocEncoding for backward compatibility.
	userpw := ctx.UserPW
	//fmt.Printf("U userpw=ctx.UserPW=%s\n", userpw)

	e := ctx.E

	key = encKey(userpw, e)

	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	switch e.R {

	case 2:
		// 4b
		u = make([]byte, 32)
		copy(u, pad)
		c.XORKeyStream(u, u)

	case 3, 4:
		// 5b
		h := md5.New()
		h.Reset()
		h.Write(pad)

		// 5c
		h.Write(e.ID)
		u = h.Sum(nil)

		// 5ds
		c.XORKeyStream(u, u)

		// 5e
		for i := 1; i <= 19; i++ {
			keynew := make([]byte, len(key))
			copy(keynew, key)

			for j := range keynew {
				keynew[j] ^= byte(i)
			}

			c, err = rc4.NewCipher(keynew)
			if err != nil {
				return nil, nil, err
			}
			c.XORKeyStream(u, u)
		}
	}

	if len(u) < 32 {
		u = append(u, nullPad32[:32-len(u)]...)
	}

	return u, key, nil
}

func validationSalt(bb []byte) []byte {
	return bb[32:40]
}

func keySalt(bb []byte) []byte {
	return bb[40:]
}

func validateOwnerPasswordAES256(ctx *Context) (ok bool, err error) {

	if len(ctx.OwnerPW) == 0 {
		return false, nil
	}

	// TODO Process PW with SASLPrep profile (RFC 4013) of stringprep (RFC 3454).
	opw := []byte(ctx.OwnerPW)
	if len(opw) > 127 {
		opw = opw[:127]
	}
	//fmt.Printf("opw <%s> isValidUTF8String: %t\n", opw, utf8.Valid(opw))

	// Algorithm 3.2a 3.
	b := append(opw, validationSalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	s := sha256.Sum256(b)

	if !bytes.HasPrefix(ctx.E.O, s[:]) {
		return false, nil
	}

	b = append(opw, keySalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	key := sha256.Sum256(b)

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return false, err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.OE)

	return true, nil
}

func validateUserPasswordAES256(ctx *Context) (ok bool, err error) {

	// TODO Process PW with SASLPrep profile (RFC 4013) of stringprep (RFC 3454).
	upw := []byte(ctx.UserPW)
	if len(upw) > 127 {
		upw = upw[:127]
	}
	//fmt.Printf("upw <%s> isValidUTF8String: %t\n", upw, utf8.Valid(upw))

	// Algorithm 3.2a 4,
	s := sha256.Sum256(append(upw, validationSalt(ctx.E.U)...))

	if !bytes.HasPrefix(ctx.E.U, s[:]) {
		return false, nil
	}

	key := sha256.Sum256(append(upw, keySalt(ctx.E.U)...))

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return false, err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.UE)

	return true, nil
}

// ValidateOwnerPassword validates the owner password aka change permissions password.
func validateOwnerPassword(ctx *Context) (ok bool, err error) {

	e := ctx.E

	if e.R == 5 {
		return validateOwnerPasswordAES256(ctx)
	}

	// The PW string is generated from OS codepage characters by first converting the string to
	// PDFDocEncoding. If input is Unicode, first convert to a codepage encoding , and then to
	// PDFDocEncoding for backward compatibility.

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	// 7a: Alg.3 p62 a-d
	key := key(ownerpw, userpw, e.R, e.L)

	// 7b
	upw := make([]byte, len(e.O))
	copy(upw, e.O)

	var c *rc4.Cipher

	switch e.R {

	case 2:
		c, err = rc4.NewCipher(key)
		if err != nil {
			return
		}
		c.XORKeyStream(upw, upw)

	case 3, 4:
		for i := 19; i >= 0; i-- {

			keynew := make([]byte, len(key))
			copy(keynew, key)

			for j := range keynew {
				keynew[j] ^= byte(i)
			}

			c, err = rc4.NewCipher(keynew)
			if err != nil {
				return false, err
			}

			c.XORKeyStream(upw, upw)
		}
	}

	// Save user pw
	upws := ctx.UserPW

	ctx.UserPW = string(upw)
	ok, err = validateUserPassword(ctx)

	// Restore user pw
	ctx.UserPW = upws

	return ok, err
}

// SupportedCFEntry returns true if all entries found are supported.
func supportedCFEntry(d Dict) (bool, error) {

	cfm := d.NameEntry("CFM")
	if cfm != nil && *cfm != "V2" && *cfm != "AESV2" && *cfm != "AESV3" {
		return false, errors.New("pdfcpu: supportedCFEntry: invalid entry \"CFM\"")
	}

	ae := d.NameEntry("AuthEvent")
	if ae != nil && *ae != "DocOpen" {
		return false, errors.New("pdfcpu: supportedCFEntry: invalid entry \"AuthEvent\"")
	}

	l := d.IntEntry("Length")
	if l != nil && (*l < 5 || *l > 16) && *l != 32 {
		return false, errors.New("pdfcpu: supportedCFEntry: invalid entry \"Length\"")
	}

	return cfm != nil && (*cfm == "AESV2" || *cfm == "AESV3"), nil
}

func perms(p int) (list []string) {

	list = append(list, fmt.Sprintf("permission bits: %12b", uint32(p)&0x0F3C))
	list = append(list, fmt.Sprintf("Bit  3: %t (print(rev2), print quality(rev>=3))", p&0x0004 > 0))
	list = append(list, fmt.Sprintf("Bit  4: %t (modify other than controlled by bits 6,9,11)", p&0x0008 > 0))
	list = append(list, fmt.Sprintf("Bit  5: %t (extract(rev2), extract other than controlled by bit 10(rev>=3))", p&0x0010 > 0))
	list = append(list, fmt.Sprintf("Bit  6: %t (add or modify annotations)", p&0x0020 > 0))
	list = append(list, fmt.Sprintf("Bit  9: %t (fill in form fields(rev>=3)", p&0x0100 > 0))
	list = append(list, fmt.Sprintf("Bit 10: %t (extract(rev>=3))", p&0x0200 > 0))
	list = append(list, fmt.Sprintf("Bit 11: %t (modify(rev>=3))", p&0x0400 > 0))
	list = append(list, fmt.Sprintf("Bit 12: %t (print high-level(rev>=3))", p&0x0800 > 0))

	return list
}

// Permissions returns a list of set permissions.
func Permissions(ctx *Context) (list []string) {

	if ctx.E == nil {
		return append(list, "Full access")
	}

	return perms(ctx.E.P)
}

func validatePermissions(ctx *Context) (bool, error) {

	// Algorithm 3.2a 5.

	if ctx.E.R != 5 {
		return true, nil
	}

	cb, err := aes.NewCipher(ctx.EncKey[:])
	if err != nil {
		return false, err
	}

	p := make([]byte, len(ctx.E.Perms))
	cb.Decrypt(p, ctx.E.Perms)
	if string(p[9:12]) != "adb" {
		return false, nil
	}

	b := binary.LittleEndian.Uint32(p[:4])
	return int32(b) == int32(ctx.E.P), nil
}

func writePermissions(ctx *Context, d Dict) error {

	// Algorithm 3.10

	if ctx.E.R != 5 {
		return nil
	}

	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, uint64(ctx.E.P))

	b[4] = 0xFF
	b[5] = 0xFF
	b[6] = 0xFF
	b[7] = 0xFF

	var c byte = 'F'
	if ctx.E.Emd {
		c = 'T'
	}
	b[8] = c

	b[9] = 'a'
	b[10] = 'd'
	b[11] = 'b'

	cb, err := aes.NewCipher(ctx.EncKey[:])
	if err != nil {
		return err
	}

	cb.Encrypt(ctx.E.Perms, b)
	d.Update("Perms", HexLiteral(hex.EncodeToString(ctx.E.Perms)))

	return nil
}

func logP(enc *Enc) {

	for _, s := range perms(enc.P) {
		log.Info.Println(s)
	}

}

func maskExtract(mode CommandMode, secHandlerRev int) int {

	p, ok := perm[mode]

	// no permissions defined or don't need extract permission
	if !ok || p.extract == 0 {
		return 0
	}

	// need extract permission

	if secHandlerRev >= 3 {
		return 0x0200 // need bit 10
	}

	return 0x0010 // need bit 5
}

func maskModify(mode CommandMode, secHandlerRev int) int {

	p, ok := perm[mode]

	// no permissions defined or don't need modify permission
	if !ok || p.modify == 0 {
		return 0
	}

	// need modify permission

	if secHandlerRev >= 3 {
		return 0x0400 // need bit 11
	}

	return 0x0008 // need bit 4
}

// HasNeededPermissions returns true if permissions for pdfcpu processing are present.
func hasNeededPermissions(mode CommandMode, enc *Enc) bool {

	// see 7.6.3.2

	logP(enc)

	m := maskExtract(mode, enc.R)
	if m > 0 {
		if enc.P&m == 0 {
			return false
		}
	}

	m = maskModify(mode, enc.R)
	if m > 0 {
		if enc.P&m == 0 {
			return false
		}
	}

	return true
}

func getV(d Dict) (*int, error) {

	v := d.IntEntry("V")

	if v == nil || (*v != 1 && *v != 2 && *v != 4 && *v != 5) {
		return nil, errors.Errorf("getV: \"V\" must be one of 1,2,4,5")
	}

	return v, nil
}
func checkStmf(ctx *Context, stmf *string, cfDict Dict) error {

	if stmf != nil && *stmf != "Identity" {

		d := cfDict.DictEntry(*stmf)
		if d == nil {
			return errors.Errorf("pdfcpu: checkStmf: entry \"%s\" missing in \"CF\"", *stmf)
		}

		aes, err := supportedCFEntry(d)
		if err != nil {
			return errors.Wrapf(err, "pdfcpu: checkStmv: unsupported \"%s\" entry in \"CF\"", *stmf)
		}
		ctx.AES4Streams = aes
	}

	return nil
}

func checkV(ctx *Context, d Dict) (*int, error) {

	v, err := getV(d)
	if err != nil {
		return nil, err
	}

	// v == 2 implies RC4
	if *v != 4 && *v != 5 {
		return v, nil
	}

	// CF
	cfDict := d.DictEntry("CF")
	if cfDict == nil {
		return nil, errors.Errorf("pdfcpu: checkV: required entry \"CF\" missing.")
	}

	// StmF
	stmf := d.NameEntry("StmF")
	err = checkStmf(ctx, stmf, cfDict)
	if err != nil {
		return nil, err
	}

	// StrF
	strf := d.NameEntry("StrF")
	if strf != nil && *strf != "Identity" {
		d1 := cfDict.DictEntry(*strf)
		if d1 == nil {
			return nil, errors.Errorf("pdfcpu: checkV: entry \"%s\" missing in \"CF\"", *strf)
		}
		aes, err := supportedCFEntry(d1)
		if err != nil {
			return nil, errors.Wrapf(err, "checkV: unsupported \"%s\" entry in \"CF\"", *strf)
		}
		ctx.AES4Strings = aes
	}

	// EFF
	eff := d.NameEntry("EFF")
	if eff != nil && *eff != "Identity" {
		d := cfDict.DictEntry(*eff)
		if d == nil {
			return nil, errors.Errorf("pdfcpu: checkV: entry \"%s\" missing in \"CF\"", *eff)
		}
		aes, err := supportedCFEntry(d)
		if err != nil {
			return nil, errors.Wrapf(err, "checkV: unsupported \"%s\" entry in \"CF\"", *eff)
		}
		ctx.AES4EmbeddedStreams = aes
	}

	return v, nil
}

func length(d Dict) (int, error) {

	l := d.IntEntry("Length")
	if l == nil {
		return 40, nil
	}

	if (*l < 40 || *l > 128 || *l%8 > 0) && *l != 256 {
		return 0, errors.Errorf("pdfcpu: length: \"Length\" %d not supported\n", *l)
	}

	return *l, nil
}

func getR(d Dict) (int, error) {

	r := d.IntEntry("R")
	if r == nil || *r < 2 || *r > 5 {
		if r != nil && *r > 5 {
			return 0, errors.New("pdfcpu: PDF 2.0 encryption not supported")
		}
		return 0, errors.New("pdfcpu: encryption: \"R\" must be 2,3,4,5")
	}

	return *r, nil
}

func validateAlgorithm(ctx *Context) (ok bool) {

	k := ctx.EncryptKeyLength

	if ctx.EncryptUsingAES {
		return k == 40 || k == 128 || k == 256
	}

	return k == 40 || k == 128
}

func validateAES256Parameters(d Dict) (oe, ue, perms []byte, err error) {

	for {

		// OE
		oe, err = d.StringEntryBytes("OE")
		if err != nil {
			break
		}
		if oe == nil || len(oe) != 32 {
			err = errors.New("pdfcpu: unsupported encryption: required entry \"OE\" missing or invalid")
			break
		}

		// UE
		ue, err = d.StringEntryBytes("UE")
		if err != nil {
			break
		}
		if ue == nil || len(ue) != 32 {
			err = errors.New("pdfcpu: unsupported encryption: required entry \"UE\" missing or invalid")
			break
		}

		// Perms
		perms, err = d.StringEntryBytes("Perms")
		if err != nil {
			break
		}
		if perms == nil || len(perms) != 16 {
			err = errors.New("pdfcpu: unsupported encryption: required entry \"Perms\" missing or invalid")
		}

		break
	}

	return oe, ue, perms, err
}

func validateOAndU(d Dict) (o, u []byte, err error) {

	for {

		// O
		o, err = d.StringEntryBytes("O")
		if err != nil {
			break
		}
		if o == nil || len(o) != 32 && len(o) != 48 {
			err = errors.New("pdfcpu: unsupported encryption: missing or invalid required entry \"O\"")
			break
		}

		// U
		u, err = d.StringEntryBytes("U")
		if err != nil {
			break
		}
		if u == nil || len(u) != 32 && len(u) != 48 {
			err = errors.New("pdfcpu: unsupported encryption: missing or invalid required entry \"U\"")
		}

		break
	}

	return o, u, err
}

// SupportedEncryption returns a pointer to a struct encapsulating used encryption.
func supportedEncryption(ctx *Context, d Dict) (*Enc, error) {

	// Filter
	filter := d.NameEntry("Filter")
	if filter == nil || *filter != "Standard" {
		return nil, errors.New("pdfcpu: unsupported encryption: filter must be \"Standard\"")
	}

	// SubFilter
	if d.NameEntry("SubFilter") != nil {
		return nil, errors.New("pdfcpu: unsupported encryption: \"SubFilter\" not supported")
	}

	// V
	v, err := checkV(ctx, d)
	if err != nil {
		return nil, err
	}

	// Length
	l, err := length(d)
	if err != nil {
		return nil, err
	}

	// R
	r, err := getR(d)
	if err != nil {
		return nil, err
	}

	o, u, err := validateOAndU(d)
	if err != nil {
		return nil, err
	}

	var oe, ue, perms []byte
	if r == 5 {
		oe, ue, perms, err = validateAES256Parameters(d)
		if err != nil {
			return nil, err
		}
	}

	// P
	p := d.IntEntry("P")
	if p == nil {
		return nil, errors.New("pdfcpu: unsupported encryption: required entry \"P\" missing")
	}

	// EncryptMetadata
	encMeta := true
	emd := d.BooleanEntry("EncryptMetadata")
	if emd != nil {
		encMeta = *emd
	}

	return &Enc{
			O:     o,
			OE:    oe,
			U:     u,
			UE:    ue,
			L:     l,
			P:     *p,
			Perms: perms,
			R:     r,
			V:     *v,
			Emd:   encMeta},
		nil
}

func decryptKey(objNumber, generation int, key []byte, aes bool) []byte {

	m := md5.New()

	nr := uint32(objNumber)
	b1 := []byte{byte(nr), byte(nr >> 8), byte(nr >> 16)}
	b := append(key, b1...)

	gen := uint16(generation)
	b2 := []byte{byte(gen), byte(gen >> 8)}
	b = append(b, b2...)

	m.Write(b)

	if aes {
		m.Write([]byte("sAlT"))
	}

	dk := m.Sum(nil)

	l := len(key) + 5
	if l < 16 {
		dk = dk[:l]
	}

	return dk
}

// EncryptBytes encrypts s using RC4 or AES.
func encryptBytes(b []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {

	if needAES {
		k := encKey
		if r != 5 {
			k = decryptKey(objNr, genNr, encKey, needAES)
		}
		bb, err := encryptAESBytes(b, k)
		if err != nil {
			return nil, err
		}
		return bb, nil
	}

	return applyRC4CipherBytes(b, objNr, genNr, encKey, needAES)
}

// EncryptString encrypts s using RC4 or AES.
func encryptString(s string, objNr, genNr int, key []byte, needAES bool, r int) (*string, error) {

	b, err := encryptBytes([]byte(s), objNr, genNr, key, needAES, r)
	if err != nil {
		return nil, err
	}

	s1, err := Escape(string(b))
	if err != nil {
		return nil, err
	}

	return s1, err
}

// decryptBytes decrypts bb using RC4 or AES.
func decryptBytes(b []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {

	if needAES {
		k := encKey
		if r != 5 {
			k = decryptKey(objNr, genNr, encKey, needAES)
		}
		bb, err := decryptAESBytes(b, k)
		if err != nil {
			return nil, err
		}
		return bb, nil
	}

	return applyRC4CipherBytes(b, objNr, genNr, encKey, needAES)
}

// decryptString decrypts s using RC4 or AES.
func decryptString(s string, objNr, genNr int, key []byte, needAES bool, r int) ([]byte, error) {

	bb, err := Unescape(s)
	if err != nil {
		return nil, err
	}

	return decryptBytes(bb, objNr, genNr, key, needAES, r)
}

func applyRC4CipherBytes(b []byte, objNr, genNr int, key []byte, needAES bool) ([]byte, error) {

	c, err := rc4.NewCipher(decryptKey(objNr, genNr, key, needAES))
	if err != nil {
		return nil, err
	}

	c.XORKeyStream(b, b)

	return b, nil
}

func encrypt(m map[string]Object, k string, v Object, objNr, genNr int, key []byte, needAES bool, r int) error {

	s, err := encryptDeepObject(v, objNr, genNr, key, needAES, r)
	if err != nil {
		return err
	}

	if s != nil {
		m[k] = *s
	}

	return nil
}

func encryptDict(d Dict, objNr, genNr int, key []byte, needAES bool, r int) error {

	for k, v := range d {
		err := encrypt(d, k, v, objNr, genNr, key, needAES, r)
		if err != nil {
			return err
		}
	}

	return nil
}

// EncryptDeepObject recurses over non trivial PDF objects and encrypts all strings encountered.
func encryptDeepObject(objIn Object, objNr, genNr int, key []byte, needAES bool, r int) (*HexLiteral, error) {

	_, ok := objIn.(IndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case StreamDict:
		err := encryptDict(obj.Dict, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}

	case Dict:
		err := encryptDict(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}

	case Array:
		for i, v := range obj {
			s, err := encryptDeepObject(v, objNr, genNr, key, needAES, r)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case StringLiteral:
		s := obj.Value()
		b, err := encryptBytes([]byte(s), objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		hl := NewHexLiteral(b)
		return &hl, nil

	case HexLiteral:
		bb, err := encryptHexLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		hl := NewHexLiteral(bb)
		return &hl, nil

	default:

	}

	return nil, nil
}

func decryptDeepObject(objIn Object, objNr, genNr int, key []byte, needAES bool, r int) (*HexLiteral, error) {

	_, ok := objIn.(IndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case Dict:
		for k, v := range obj {
			s, err := decryptDeepObject(v, objNr, genNr, key, needAES, r)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[k] = *s
			}
		}

	case Array:
		for i, v := range obj {
			s, err := decryptDeepObject(v, objNr, genNr, key, needAES, r)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case StringLiteral:
		bb, err := decryptString(obj.Value(), objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		hl := NewHexLiteral(bb)
		return &hl, nil

	case HexLiteral:
		bb, err := decryptHexLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		hl := NewHexLiteral(bb)
		return &hl, nil

	default:

	}

	return nil, nil
}

// EncryptStream encrypts a stream buffer using RC4 or AES.
func encryptStream(buf []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {

	k := encKey
	if r != 5 {
		k = decryptKey(objNr, genNr, encKey, needAES)
	}

	if needAES {
		return encryptAESBytes(buf, k)
	}

	return applyRC4Bytes(buf, k)
}

// decryptStream decrypts a stream buffer using RC4 or AES.
func decryptStream(buf []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {

	k := encKey
	if r != 5 {
		k = decryptKey(objNr, genNr, encKey, needAES)
	}

	if needAES {
		return decryptAESBytes(buf, k)
	}

	return applyRC4Bytes(buf, k)
}

func applyRC4Bytes(buf, key []byte) ([]byte, error) {

	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer

	r := &cipher.StreamReader{S: c, R: bytes.NewReader(buf)}

	_, err = io.Copy(&b, r)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func encryptAESBytes(b, key []byte) ([]byte, error) {

	// pad b to aes.Blocksize
	l := len(b) % aes.BlockSize
	c := 0x10
	if l > 0 {
		c = aes.BlockSize - l
	}
	b = append(b, bytes.Repeat([]byte{byte(c)}, aes.BlockSize-l)...)

	if len(b) < aes.BlockSize {
		return nil, errors.New("pdfcpu: encryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("pdfcpu: encryptAESBytes: Ciphertext not a multiple of block size")
	}

	data := make([]byte, aes.BlockSize+len(b))
	iv := data[:aes.BlockSize]

	_, err := io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(data[aes.BlockSize:], b)

	return data, nil
}

func decryptAESBytes(b, key []byte) ([]byte, error) {

	if len(b) < aes.BlockSize {
		return nil, errors.New("pdfcpu: decryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("pdfcpu: decryptAESBytes: Ciphertext not a multiple of block size")
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	copy(iv, b[:aes.BlockSize])

	data := b[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(data, data)

	// Remove padding.
	// Note: For some reason not all AES ciphertexts are padded.
	if len(data) > 0 && data[len(data)-1] <= 0x10 {
		e := len(data) - int(data[len(data)-1])
		data = data[:e]
	}

	return data, nil
}

func fileID(ctx *Context) (HexLiteral, error) {

	// see also 14.4 File Identifiers.

	// The calculation of the file identifier need not be reproducible;
	// all that matters is that the identifier is likely to be unique.
	// For example, two implementations of the preceding algorithm might use different formats for the current time,
	// causing them to produce different file identifiers for the same file created at the same time,
	// but the uniqueness of the identifier is not affected.

	h := md5.New()

	// Current timestamp.
	h.Write([]byte(time.Now().String()))

	// File location - ignore, we don't have this.

	// File size.
	h.Write([]byte(strconv.Itoa(ctx.Read.ReadFileSize())))

	// All values of the info dict which is assumed to be there at this point.
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		return "", err
	}

	for _, v := range d {
		o, err := ctx.Dereference(v)
		if err != nil {
			return "", err
		}
		h.Write([]byte(o.String()))
	}

	m := h.Sum(nil)

	return HexLiteral(hex.EncodeToString(m)), nil
}

func encryptHexLiteral(hl HexLiteral, objNr, genNr int, key []byte, needAES bool, r int) ([]byte, error) {

	bb, err := hl.Bytes()
	if err != nil {
		return nil, err
	}

	return encryptBytes(bb, objNr, genNr, key, needAES, r)
}

func decryptHexLiteral(hl HexLiteral, objNr, genNr int, key []byte, needAES bool, r int) ([]byte, error) {

	bb, err := hl.Bytes()
	if err != nil {
		return nil, err
	}

	return decryptBytes(bb, objNr, genNr, key, needAES, r)
}

func calcFileEncKeyFromUE(ctx *Context) (k []byte, err error) {

	upw := []byte(ctx.OwnerPW)
	key := sha256.Sum256(append(upw, keySalt(ctx.E.U)...))

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	iv := make([]byte, 16)
	k = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(k, ctx.E.UE)

	return k, nil
}

func calcFileEncKeyFromOE(ctx *Context) (k []byte, err error) {

	opw := []byte(ctx.OwnerPW)
	b := append(opw, keySalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	key := sha256.Sum256(b)

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	iv := make([]byte, 16)
	k = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(k, ctx.E.OE)

	return k, nil
}

func calcFileEncKey(ctx *Context, d Dict) (err error) {

	// Calc Random UE (32 bytes)
	ue := make([]byte, 32)
	_, err = io.ReadFull(rand.Reader, ue)
	if err != nil {
		return err
	}

	ctx.E.UE = ue
	d.Update("UE", HexLiteral(hex.EncodeToString(ctx.E.UE)))

	// Calc file encryption key.
	ctx.EncKey, err = calcFileEncKeyFromUE(ctx)

	return err
}

func calcOAndUAES256(ctx *Context, d Dict) (err error) {

	// 1) Calc U.
	b := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return err
	}

	u := append(make([]byte, 32), b...)
	upw := []byte(ctx.UserPW)
	h := sha256.Sum256(append(upw, validationSalt(u)...))
	ctx.E.U = append(h[:], b...)
	d.Update("U", HexLiteral(hex.EncodeToString(ctx.E.U)))

	// 2) Calc O (depends on U).
	b = make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return err
	}

	o := append(make([]byte, 32), b...)
	opw := []byte(ctx.OwnerPW)
	c := append(opw, validationSalt(o)...)
	h = sha256.Sum256(append(c, ctx.E.U...))
	ctx.E.O = append(h[:], b...)
	d.Update("O", HexLiteral(hex.EncodeToString(ctx.E.O)))

	err = calcFileEncKey(ctx, d)
	if err != nil {
		return err
	}

	// Encrypt file encryption key into UE.
	h = sha256.Sum256(append(upw, keySalt(u)...))
	cb, err := aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	mode := cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.UE, ctx.EncKey)
	d.Update("UE", HexLiteral(hex.EncodeToString(ctx.E.UE)))

	// Encrypt file encryption key into OE.
	c = append(opw, keySalt(o)...)
	h = sha256.Sum256(append(c, ctx.E.U...))
	cb, err = aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	mode = cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.OE, ctx.EncKey)
	d.Update("OE", HexLiteral(hex.EncodeToString(ctx.E.OE)))

	return nil
}

func calcOAndU(ctx *Context, d Dict) (err error) {

	if ctx.E.R == 5 {
		return calcOAndUAES256(ctx, d)
	}

	ctx.E.O, err = o(ctx)
	if err != nil {
		return err
	}

	ctx.E.U, ctx.EncKey, err = u(ctx)
	if err != nil {
		return err
	}

	d.Update("U", HexLiteral(hex.EncodeToString(ctx.E.U)))
	d.Update("O", HexLiteral(hex.EncodeToString(ctx.E.O)))

	return nil
}
