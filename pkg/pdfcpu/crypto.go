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
	"crypto/sha512"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/text/secure/precis"
	"golang.org/x/text/unicode/norm"
)

var (
	errAESCiphertextTooShort  = errors.New("ciphertext too short")
	errAESCiphertextUnaligned = errors.New("ciphertext not a multiple of block size")

	pad = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}

	nullPad32 = make([]byte, 32)

	// Needed permission bits for pdfcpu commands.
	perm = map[model.CommandMode]struct{ extract, modify int }{
		model.VALIDATE:                {0, 0},
		model.LISTINFO:                {0, 0},
		model.OPTIMIZE:                {0, 0},
		model.SPLIT:                   {1, 0},
		model.SPLITBYPAGENR:           {1, 0},
		model.MERGECREATE:             {0, 0},
		model.MERGECREATEZIP:          {0, 0},
		model.MERGEAPPEND:             {0, 0},
		model.EXTRACTIMAGES:           {1, 0},
		model.EXTRACTFONTS:            {1, 0},
		model.EXTRACTPAGES:            {1, 0},
		model.EXTRACTCONTENT:          {1, 0},
		model.EXTRACTMETADATA:         {1, 0},
		model.TRIM:                    {0, 1},
		model.LISTATTACHMENTS:         {0, 0},
		model.EXTRACTATTACHMENTS:      {1, 0},
		model.ADDATTACHMENTS:          {0, 1},
		model.ADDATTACHMENTSPORTFOLIO: {0, 1},
		model.REMOVEATTACHMENTS:       {0, 1},
		model.LISTPERMISSIONS:         {0, 0},
		model.SETPERMISSIONS:          {0, 0},
		model.ADDWATERMARKS:           {0, 1},
		model.REMOVEWATERMARKS:        {0, 1},
		model.IMPORTIMAGES:            {0, 1},
		model.INSERTPAGESBEFORE:       {0, 1},
		model.INSERTPAGESAFTER:        {0, 1},
		model.REMOVEPAGES:             {0, 1},
		model.LISTKEYWORDS:            {0, 0},
		model.ADDKEYWORDS:             {0, 1},
		model.REMOVEKEYWORDS:          {0, 1},
		model.LISTPROPERTIES:          {0, 0},
		model.ADDPROPERTIES:           {0, 1},
		model.REMOVEPROPERTIES:        {0, 1},
		model.COLLECT:                 {1, 0},
		model.CROP:                    {0, 1},
		model.LISTBOXES:               {0, 0},
		model.ADDBOXES:                {0, 1},
		model.REMOVEBOXES:             {0, 1},
		model.LISTANNOTATIONS:         {0, 1},
		model.ADDANNOTATIONS:          {0, 1},
		model.REMOVEANNOTATIONS:       {0, 1},
		model.ROTATE:                  {0, 1},
		model.NUP:                     {0, 1},
		model.GRID:                    {0, 1},
		model.BOOKLET:                 {0, 1},
		model.LISTBOOKMARKS:           {0, 0},
		model.ADDBOOKMARKS:            {0, 1},
		model.REMOVEBOOKMARKS:         {0, 1},
		model.IMPORTBOOKMARKS:         {0, 1},
		model.EXPORTBOOKMARKS:         {0, 1},
		model.LISTIMAGES:              {0, 1},
		model.UPDATEIMAGES:            {0, 1},
		model.CREATE:                  {0, 0},
		model.DUMP:                    {0, 1},
		model.LISTFORMFIELDS:          {0, 0},
		model.REMOVEFORMFIELDS:        {0, 1},
		model.LOCKFORMFIELDS:          {0, 1},
		model.UNLOCKFORMFIELDS:        {0, 1},
		model.RESETFORMFIELDS:         {0, 1},
		model.EXPORTFORMFIELDS:        {0, 1},
		model.FILLFORMFIELDS:          {0, 1},
		model.LISTPAGELAYOUT:          {0, 1},
		model.SETPAGELAYOUT:           {0, 1},
		model.RESETPAGELAYOUT:         {0, 1},
		model.LISTPAGEMODE:            {0, 1},
		model.SETPAGEMODE:             {0, 1},
		model.RESETPAGEMODE:           {0, 1},
		model.LISTVIEWERPREFERENCES:   {0, 1},
		model.SETVIEWERPREFERENCES:    {0, 1},
		model.RESETVIEWERPREFERENCES:  {0, 1},
		model.ZOOM:                    {0, 1},
	}

	// ErrMalformedEncryption reports a missing, malformed, or inconsistent encryption dictionary.
	ErrMalformedEncryption = errors.New("malformed encryption")

	// ErrUnsupportedEncryptionFeature reports a recognized encryption algorithm, version, or security handler that pdfcpu cannot process.
	ErrUnsupportedEncryptionFeature = errors.New("unsupported encryption feature")
)

func classifyEncryptionDictionaryError(err error) error {
	if errors.Is(err, errUnregisteredObject) ||
		errors.Is(err, errNilDereferencedObject) ||
		errors.Is(err, ErrReferenceDoesNotExist) ||
		errors.Is(err, errCorruptDictObject) ||
		errors.Is(err, model.ErrMissingEncryptDictObject) ||
		errors.Is(err, model.ErrWrongTypeEncryptDictObject) ||
		errors.Is(err, model.ErrDictionaryCorrupt) {
		return errors.Join(ErrMalformedEncryption, err)
	}
	return err
}

// NewEncryptDict creates a new EncryptDict using the standard security handler.
func newEncryptDict(pdf20, needAES bool, keyLength int, permissions int16) types.Dict {
	d := types.NewDict()

	d.Insert("Filter", types.Name("Standard"))

	if keyLength >= 128 {
		d.Insert("Length", types.Integer(keyLength))
		i := 4
		if keyLength == 256 {
			// PDF 2.0 AES-256 uses V=5/R=6 with AESV3.
			// TODO Support ISO/TS 32003 V=6/AESV4 AES-256-GCM.
			i = 5
		}
		d.Insert("V", types.Integer(i))
		if pdf20 {
			i++
		}
		d.Insert("R", types.Integer(i))
	} else {
		d.Insert("R", types.Integer(2))
		d.Insert("V", types.Integer(1))
	}

	// Set user access permission flags.
	d.Insert("P", types.Integer(permissions))

	if keyLength == 128 || keyLength == 256 {
		d1 := types.NewDict()
		d1.Insert("AuthEvent", types.Name("DocOpen"))
		cfm := "V2"
		if needAES {
			if keyLength == 128 {
				cfm = "AESV2"
			}
			if keyLength == 256 {
				cfm = "AESV3"
			}
		}
		d1.Insert("CFM", types.Name(cfm))
		kl := keyLength
		if pdf20 {
			kl /= 8
		}
		d1.Insert("Length", types.Integer(kl))

		d2 := types.NewDict()
		d2.Insert("StdCF", d1)
		d.Insert("CF", d2)
		d.Insert("StmF", types.Name("StdCF"))
		d.Insert("StrF", types.Name("StdCF"))
	}

	if keyLength == 256 {
		d.Insert("U", types.NewHexLiteral(make([]byte, 48)))
		d.Insert("O", types.NewHexLiteral(make([]byte, 48)))
		d.Insert("UE", types.NewHexLiteral(make([]byte, 32)))
		d.Insert("OE", types.NewHexLiteral(make([]byte, 32)))
		d.Insert("Perms", types.NewHexLiteral(make([]byte, 16)))
	} else {
		d.Insert("U", types.NewHexLiteral(make([]byte, 32)))
		d.Insert("O", types.NewHexLiteral(make([]byte, 32)))
	}

	return d
}

func encKey(userpw string, e *model.Enc) (key []byte) {
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
		for range 50 {
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
func validateUserPassword(ctx *model.Context) (ok bool, err error) {
	if ctx.E.R == 5 {
		return validateUserPasswordAES256(ctx)
	}

	if ctx.E.R == 6 {
		return validateUserPasswordAES256Rev6(ctx)
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
		ok = passwordHashEqual(ctx.E.U, u)

	case 3, 4:
		ok = passwordHashPrefixEqual(ctx.E.U, u[:16])
	}

	return ok, nil
}

func passwordHashEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func passwordHashPrefixEqual(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	return passwordHashEqual(b[:len(prefix)], prefix)
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
func o(ctx *model.Context) ([]byte, error) {
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
func u(ctx *model.Context) (u []byte, key []byte, err error) {
	// The PW string is generated from OS codepage characters by first converting the string to PDFDocEncoding.
	// If input is Unicode, first convert to a codepage encoding , and then to PDFDocEncoding for backward compatibility.
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
	return bb[40:48]
}

func decryptOE(ctx *model.Context, opw []byte) error {
	b := append(opw, keySalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	key := sha256.Sum256(b)

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.OE)

	return nil
}

func validateOwnerPasswordAES256(ctx *model.Context) (ok bool, err error) {
	if len(ctx.OwnerPW) == 0 {
		return false, nil
	}

	opw, err := processInput(ctx.OwnerPW)
	if err != nil {
		return false, err
	}

	if len(opw) > 127 {
		opw = opw[:127]
	}

	// Algorithm 3.2a 3.
	b := append(opw, validationSalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	s := sha256.Sum256(b)

	if !passwordHashPrefixEqual(ctx.E.O, s[:]) {
		return false, nil
	}

	if err := decryptOE(ctx, opw); err != nil {
		return false, err
	}

	return true, nil
}

func decryptUE(ctx *model.Context, upw []byte) error {
	key := sha256.Sum256(append(upw, keySalt(ctx.E.U)...))

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.UE)

	return nil
}

func validateUserPasswordAES256(ctx *model.Context) (ok bool, err error) {
	upw, err := processInput(ctx.UserPW)
	if err != nil {
		return false, err
	}

	if len(upw) > 127 {
		upw = upw[:127]
	}

	// Algorithm 3.2a 4,
	s := sha256.Sum256(append(upw, validationSalt(ctx.E.U)...))

	if !passwordHashPrefixEqual(ctx.E.U, s[:]) {
		return false, nil
	}

	if err := decryptUE(ctx, upw); err != nil {
		return false, err
	}

	return true, nil
}

func processInput(input string) ([]byte, error) {
	// Create a new Precis profile for SASLprep
	p := precis.NewIdentifier(
		precis.BidiRule,
		precis.Norm(norm.NFKC),
	)

	output, err := p.String(input)
	if err != nil {
		return nil, err
	}

	return []byte(output), nil
}

func hashRev6(input, pw, U []byte) ([]byte, int, error) {
	// 7.6.4.3.4 Algorithm 2.B returns 32 bytes.

	mod3 := new(big.Int).SetUint64(3)

	k0 := sha256.Sum256(input)
	k := k0[:]

	var e []byte
	j := 0

	for ; j < 64 || e[len(e)-1] > byte(j-32); j++ {
		var k1 []byte
		bb := append(pw, k...)
		if len(U) > 0 {
			bb = append(bb, U...)
		}
		for i := 0; i < 64; i++ {
			k1 = append(k1, bb...)
		}

		cb, err := aes.NewCipher(k[:16])
		if err != nil {
			return nil, -1, err
		}

		iv := k[16:32]
		e = make([]byte, len(k1))
		mode := cipher.NewCBCEncrypter(cb, iv)
		mode.CryptBlocks(e, k1)

		num := new(big.Int).SetBytes(e[:16])
		r := (new(big.Int).Mod(num, mod3)).Uint64()

		switch r {
		case 0:
			k0 := sha256.Sum256(e)
			k = k0[:]
		case 1:
			k0 := sha512.Sum384(e)
			k = k0[:]
		case 2:
			k0 := sha512.Sum512(e)
			k = k0[:]
		}

	}

	return k[:32], j, nil
}

func validateOwnerPasswordAES256Rev6(ctx *model.Context) (ok bool, err error) {
	if len(ctx.OwnerPW) == 0 {
		return false, nil
	}

	// Process PW with SASLPrep profile (RFC 4013) of stringprep (RFC 3454).
	opw, err := processInput(ctx.OwnerPW)
	if err != nil {
		return false, err
	}

	if len(opw) > 127 {
		opw = opw[:127]
	}

	// Algorithm 12
	bb := append(opw, validationSalt(ctx.E.O)...)
	bb = append(bb, ctx.E.U...)
	s, _, err := hashRev6(bb, opw, ctx.E.U)
	if err != nil {
		return false, err
	}

	if !passwordHashPrefixEqual(ctx.E.O, s[:]) {
		return false, nil
	}

	bb = append(opw, keySalt(ctx.E.O)...)
	bb = append(bb, ctx.E.U...)
	key, _, err := hashRev6(bb, opw, ctx.E.U)
	if err != nil {
		return false, err
	}

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

func validateUserPasswordAES256Rev6(ctx *model.Context) (bool, error) {
	if len(ctx.E.UE) != 32 {
		return false, errors.New("UE: invalid length")
	}

	upw, err := processInput(ctx.UserPW)
	if err != nil {
		return false, err
	}
	if len(upw) > 127 {
		upw = upw[:127]
	}

	// Validate U prefix
	bb := append([]byte{}, upw...)
	bb = append(bb, validationSalt(ctx.E.U)...)
	s, _, err := hashRev6(bb, upw, nil)
	if err != nil {
		return false, err
	}
	if !passwordHashPrefixEqual(ctx.E.U, s) {
		return false, nil
	}

	// Derive decryption key
	bb = append([]byte{}, upw...)
	bb = append(bb, keySalt(ctx.E.U)...)
	key, _, err := hashRev6(bb, upw, nil)
	if err != nil {
		return false, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return false, err
	}

	iv := make([]byte, 16)
	encKey := make([]byte, 32)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(encKey, ctx.E.UE)
	ctx.EncKey = encKey

	return true, nil
}

// ValidateOwnerPassword validates the owner password aka change permissions password.
func validateOwnerPassword(ctx *model.Context) (ok bool, err error) {
	e := ctx.E

	if e.R == 5 {
		return validateOwnerPasswordAES256(ctx)
	}

	if e.R == 6 {
		return validateOwnerPasswordAES256Rev6(ctx)
	}

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

func perms(p int) (list []string) {
	list = append(list, fmt.Sprintf("permission bits: %012b (x%03X)", uint32(p)&0x0F3C, uint32(p)&0x0F3C))
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

// PermissionsList returns a list of set permissions.
func PermissionsList(p int) (list []string) {
	if p == 0 {
		return append(list, "Full access")
	}

	return perms(p)
}

func permissionInt32(p int) (int32, error) {
	const (
		minPermission = int64(-1 << 31)
		maxPermission = int64(1<<31 - 1)
	)

	if int64(p) < minPermission || int64(p) > maxPermission {
		return 0, fmt.Errorf("%w: permission value P %d out of signed 32-bit range", ErrMalformedEncryption, p)
	}

	return int32(p), nil
}

func normalizePermission(p int, relaxed bool) (int, error, error) {
	p32, err := permissionInt32(p)
	if err == nil {
		return int(p32), nil, nil
	}

	const maxUnsignedPermission = int64(1<<32 - 1)
	if !relaxed || p < 0 || int64(p) > maxUnsignedPermission {
		return 0, nil, err
	}

	return int(int32(uint32(p))), err, nil
}

func validatePermissions(ctx *model.Context) (bool, error) {
	// Algorithm 3.2a 5.

	if ctx.E.R != 5 && ctx.E.R != 6 {
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
	if p[8] != 'T' && p[8] != 'F' {
		return false, nil
	}
	if (p[8] == 'T') != ctx.E.Emd {
		return false, nil
	}

	b := binary.LittleEndian.Uint32(p[:4])
	expected, err := permissionInt32(ctx.E.P)
	if err != nil {
		return false, err
	}

	return b == uint32(expected), nil
}

func writePermissions(ctx *model.Context, d types.Dict) error {
	// Algorithm 3.10

	if ctx.E.R != 5 && ctx.E.R != 6 {
		return nil
	}

	p, err := permissionInt32(ctx.E.P)
	if err != nil {
		return err
	}

	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b, uint32(p))

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
	d.Update("Perms", types.HexLiteral(hex.EncodeToString(ctx.E.Perms)))

	return nil
}

func logP(enc *model.Enc) {
	if !log.InfoEnabled() {
		return
	}
	for _, s := range perms(enc.P) {
		log.Info.Println(s)
	}

}

func maskExtract(mode model.CommandMode, secHandlerRev int) int {
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

func maskModify(mode model.CommandMode, secHandlerRev int) int {
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
func hasNeededPermissions(mode model.CommandMode, enc *model.Enc) bool {
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

func getR(ctx *model.Context, d types.Dict) (int, error) {
	maxR := 6
	if ctx.XRefTable.Version() == model.V20 || ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
		maxR = 7
	}

	r := d.IntEntry("R")
	if r == nil {
		return 0, fmt.Errorf("%w: required entry \"R\" missing", ErrMalformedEncryption)
	}
	if *r < 2 || *r > maxR {
		return 0, fmt.Errorf("%w: invalid encrypt \"R\" %d", ErrMalformedEncryption, *r)
	}
	if *r == 7 {
		return 0, fmt.Errorf("%w: encrypt \"R\" 7", ErrUnsupportedEncryptionFeature)
	}

	return *r, nil
}

func validateAlgorithm(ctx *model.Context) (ok bool) {
	k := ctx.EncryptKeyLength

	if ctx.XRefTable.Version() == model.V20 {
		return ctx.EncryptUsingAES && k == 256
	}

	if ctx.EncryptUsingAES {
		return k == 40 || k == 128 || k == 256
	}

	return k == 40 || k == 128
}

func validateAES256Parameters(d types.Dict, r int) (oe, ue, perms []byte, err error) {
	if !(r == 5 || r == 6 || r == 7) {
		return nil, nil, nil, nil
	}

	// OE
	oe, err = d.StringEntryBytes("OE")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrMalformedEncryption, err)
	}
	if len(oe) != 32 {
		return nil, nil, nil, fmt.Errorf("%w: required entry \"OE\" missing or not 32 bytes", ErrMalformedEncryption)
	}

	// UE
	ue, err = d.StringEntryBytes("UE")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrMalformedEncryption, err)
	}
	if len(ue) != 32 {
		return nil, nil, nil, fmt.Errorf("%w: required entry \"UE\" missing or not 32 bytes", ErrMalformedEncryption)
	}

	// Perms
	perms, err = d.StringEntryBytes("Perms")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrMalformedEncryption, err)
	}
	if len(perms) != 16 {
		return nil, nil, nil, fmt.Errorf("%w: required entry \"Perms\" missing or not 16 bytes", ErrMalformedEncryption)
	}

	return oe, ue, perms, nil
}

func validatePasswordEntry(
	d types.Dict,
	key string,
	minLen int,
	digestShort bool,
	specViolations *[]error,
) ([]byte, error) {
	b, err := d.StringEntryBytes(key)
	if err != nil {
		return nil, fmt.Errorf("%w: entry %q: %w", ErrMalformedEncryption, key, err)
	}
	if len(b) < minLen {
		err := fmt.Errorf("%w: required entry %q shorter than %d bytes", ErrMalformedEncryption, key, minLen)
		if !digestShort {
			return nil, err
		}
		appendSpecViolation(specViolations, err)
	}
	return b, nil
}

func validateOAndU(
	d types.Dict,
	r int,
	relaxed bool,
	specViolations *[]error,
) (o, u []byte, err error) {
	minLen := 32
	digestShort := relaxed
	if r >= 5 {
		minLen = 48
		digestShort = false
	}

	o, err = validatePasswordEntry(d, "O", minLen, digestShort, specViolations)
	if err != nil {
		return nil, nil, err
	}
	u, err = validatePasswordEntry(d, "U", minLen, digestShort, specViolations)
	if err != nil {
		return nil, nil, err
	}
	return o, u, nil
}

func checkCFLengthV2(length int, pdf20, relaxed bool) (specViolation, err error) {
	bitLen := length
	if pdf20 {
		bitLen *= 8
	}
	if bitLen >= 40 && bitLen <= 128 && bitLen%8 == 0 {
		return nil, nil
	}

	err = fmt.Errorf("%w: invalid CF length: %d", ErrMalformedEncryption, length)
	if pdf20 || !relaxed {
		return nil, err
	}

	bitLen = length * 8
	if bitLen < 40 || bitLen > 128 || bitLen%8 != 0 {
		return nil, err
	}

	return err, nil
}

func checkCFLengthAESV2(length int, pdf20, relaxed bool) (specViolation, err error) {
	const aesV2KeyLength = 128

	bitLen := length
	if pdf20 {
		bitLen *= 8
	}
	if bitLen == aesV2KeyLength {
		return nil, nil
	}

	err = fmt.Errorf("%w: invalid CF length, got %d want %d", ErrMalformedEncryption, length, aesV2KeyLength)
	if pdf20 || !relaxed || length*8 != aesV2KeyLength {
		return nil, err
	}

	return err, nil
}

func validateCFLength(length *int, cfm *string, pdf20, relaxed bool) (specViolation, err error) {
	// See table 25 Length
	// v = 4

	if length == nil || cfm == nil {
		return nil, nil
	}

	switch *cfm {
	case "V2":
		return checkCFLengthV2(*length, pdf20, relaxed)

	case "AESV2":
		return checkCFLengthAESV2(*length, pdf20, relaxed)
	}

	return nil, nil
}

func appendSpecViolation(specViolations *[]error, err error) {
	if err == nil {
		return
	}
	for _, specViolation := range *specViolations {
		if specViolation.Error() == err.Error() {
			return
		}
	}
	*specViolations = append(*specViolations, err)
}

func validateCryptFilterRecipients(ctx *model.Context, d types.Dict, cfm *string) error {
	if cfm == nil {
		return nil
	}
	switch *cfm {
	case "V2", "AESV2", "AESV3", "AESV4":
	default:
		return nil
	}
	obj, ok := d.Find("Recipients")
	if !ok {
		return fmt.Errorf("%w: crypt filter missing entry \"Recipients\"", ErrMalformedEncryption)
	}

	obj1, err := ctx.Dereference(obj)
	if err != nil {
		return fmt.Errorf("crypt filter entry \"Recipients\": %w", err)
	}

	switch v := obj1.(type) {
	case types.Array:
		if len(v) == 0 {
			return fmt.Errorf("%w: crypt filter entry \"Recipients\" is empty", ErrMalformedEncryption)
		}
	case types.StringLiteral:
		if len(v.Value()) == 0 {
			return fmt.Errorf("%w: crypt filter entry \"Recipients\" is empty", ErrMalformedEncryption)
		}
	default:
		return fmt.Errorf("%w: crypt filter entry \"Recipients\" must be array or string literal", ErrMalformedEncryption)
	}

	return nil
}

func checkCryptFilterCFM(cfm string, v int) error {
	var ss []string
	switch v {
	case 4:
		ss = []string{"V2", "AESV2"}
	case 5:
		ss = []string{"AESV3"}
	case 6:
		ss = []string{"AESV4"}
	}
	if len(ss) > 0 {
		if !types.MemberOf(cfm, ss) {
			return fmt.Errorf("%w: crypt filter invalid entry \"CFM\": %s", ErrMalformedEncryption, cfm)
		}
	}
	return nil
}

func validateCryptFilterAuthEvent(d types.Dict, allowEFOpen bool) error {
	ae := d.NameEntry("AuthEvent")
	if ae != nil && *ae != "DocOpen" && (!allowEFOpen || *ae != "EFOpen") {
		return fmt.Errorf("%w: crypt filter invalid entry \"AuthEvent\"", ErrMalformedEncryption)
	}
	return nil
}

func validateCryptFilter(
	ctx *model.Context,
	d types.Dict,
	v int,
	pubKeySecHandler,
	relaxed bool,
	allowEFOpen bool,
	specViolations *[]error,
) (bool, error) {
	// v = 4,5,6
	// 4 AESV2, V2
	// 5 AESV3
	// 6 AESV4

	cfm := d.NameEntry("CFM")
	if cfm != nil {
		if err := checkCryptFilterCFM(*cfm, v); err != nil {
			return false, err
		}
	}

	length := d.IntEntry("Length")
	pdf20 := ctx.PDF20()
	if length == nil && !pdf20 && v != 5 {
		return false, fmt.Errorf("%w: crypt filter missing entry \"Length\"", ErrMalformedEncryption)
	}
	if v == 4 {
		specViolation, err := validateCFLength(length, cfm, pdf20, relaxed)
		if err != nil {
			return false, err
		}
		appendSpecViolation(specViolations, specViolation)
	}

	if err := validateCryptFilterAuthEvent(d, allowEFOpen); err != nil {
		return false, err
	}

	if pubKeySecHandler {
		if err := validateCryptFilterRecipients(ctx, d, cfm); err != nil {
			return false, err
		}
	}

	aes := cfm != nil && (*cfm == "AESV2" || *cfm == "AESV3" || *cfm == "AESV4")

	return aes, nil
}

func locateCFEntry(
	ctx *model.Context,
	d types.Dict,
	v int,
	key string,
	pubKeySecHandler,
	relaxed bool,
	allowEFOpen bool,
	specViolations *[]error,
) (bool, error) {
	d1 := d.DictEntry(key)
	if d1 == nil {
		return false, fmt.Errorf("%w: entry \"%s\" missing in \"CF\"", ErrMalformedEncryption, key)
	}
	return validateCryptFilter(ctx, d1, v, pubKeySecHandler, relaxed, allowEFOpen, specViolations)
}

func validateStmf(
	ctx *model.Context,
	d,
	cfDict types.Dict,
	v int,
	pubKeySecHandler,
	relaxed bool,
	specViolations *[]error,
) error {
	n := d.NameEntry("StmF")
	if n != nil && *n != "Identity" {
		aes, err := locateCFEntry(ctx, cfDict, v, *n, pubKeySecHandler, relaxed, false, specViolations)
		if err != nil {
			return fmt.Errorf("encrypt dict entry \"StmF\": %w", err)
		}
		ctx.AES4Streams = aes
	}
	return nil
}

func validateStrf(
	ctx *model.Context,
	d,
	cfDict types.Dict,
	v int,
	pubKeySecHandler,
	relaxed bool,
	specViolations *[]error,
) error {
	n := d.NameEntry("StrF")
	if n != nil && *n != "Identity" {
		aes, err := locateCFEntry(ctx, cfDict, v, *n, pubKeySecHandler, relaxed, false, specViolations)
		if err != nil {
			return fmt.Errorf("encrypt dict entry \"StrF\": %w", err)
		}
		ctx.AES4Strings = aes
	}
	return nil
}

func validateEFF(
	ctx *model.Context,
	d,
	cfDict types.Dict,
	v int,
	pubKeySecHandler,
	relaxed bool,
	specViolations *[]error,
) error {
	n := d.NameEntry("EFF")
	if n != nil && *n != "Identity" {
		aes, err := locateCFEntry(ctx, cfDict, v, *n, pubKeySecHandler, relaxed, true, specViolations)
		if err != nil {
			return fmt.Errorf("encrypt dict entry \"EFF\": %w", err)
		}
		ctx.AES4EmbeddedStreams = aes
	}
	return nil
}

func validateCryptFilters(
	ctx *model.Context,
	d types.Dict,
	v int,
	pubKeySecHandler bool,
	specViolations *[]error,
) error {
	// validate CF, StmF, StrF, EFF
	// v = 4,5,6

	// CF
	cfDict := d.DictEntry("CF")
	if cfDict == nil {
		return fmt.Errorf("%w: encrypt dict, required entry \"CF\" missing", ErrMalformedEncryption)
	}

	relaxed := ctx.XRefTable.ValidationMode == model.ValidationRelaxed

	if err := validateStmf(ctx, d, cfDict, v, pubKeySecHandler, relaxed, specViolations); err != nil {
		return err
	}

	if err := validateStrf(ctx, d, cfDict, v, pubKeySecHandler, relaxed, specViolations); err != nil {
		return err
	}

	return validateEFF(ctx, d, cfDict, v, pubKeySecHandler, relaxed, specViolations)
}

func validateEncryptFilter(d types.Dict) (string, error) {
	filter := d.NameEntry("Filter")
	if filter == nil {
		return "", fmt.Errorf("%w: required entry \"Filter\" missing", ErrMalformedEncryption)
	}
	// TODO support "Adobe.PubSec"
	if !types.MemberOf(*filter, []string{"Standard"}) {
		return "", fmt.Errorf("%w: filter %s", ErrUnsupportedEncryptionFeature, *filter)
	}
	return *filter, nil
}

func validateEncryptSubFilter(d types.Dict, pubKeySecHandler bool) (string, error) {
	subFilter := d.NameEntry("SubFilter")
	if subFilter != nil && pubKeySecHandler {
		if !types.MemberOf(*subFilter, []string{"adbe.pkcs7.s3", "adbe.pkcs7.s4", "adbe.pkcs7.s5"}) {
			return "", fmt.Errorf("%w: subFilter %s", ErrUnsupportedEncryptionFeature, *subFilter)
		}
		return *subFilter, nil
	}
	return "", nil
}

func validateEncryptV(d types.Dict) (int, error) {
	v := d.IntEntry("V")
	if v == nil {
		return -1, fmt.Errorf("%w: required entry \"V\" missing", ErrMalformedEncryption)
	}

	switch *v {
	case 1, 2, 3, 4, 5:
		return *v, nil
	case 6:
		return -1, fmt.Errorf("%w: encrypt \"V\" 6 (AESV4)", ErrUnsupportedEncryptionFeature)
	default:
		return -1, fmt.Errorf("%w: invalid encrypt \"V\" %d", ErrMalformedEncryption, *v)
	}
}

func validateEncryptLength(d types.Dict, v int) (int, error) {
	switch v {
	case 1:
		return 40, nil
	case 2, 4:
		i := d.IntEntry("Length")
		if i == nil {
			return 40, nil
		}
		if *i < 40 || *i > 128 || *i%8 != 0 {
			return 0, fmt.Errorf("%w: invalid encrypt \"Length\" %d", ErrMalformedEncryption, *i)
		}
		return *i, nil
	case 5:
		return 256, nil
	case 6:
		return 0, fmt.Errorf("%w: encrypt \"V\" 6 (AESV4)", ErrUnsupportedEncryptionFeature)
	default:
		return 0, fmt.Errorf("%w: invalid encrypt \"V\" %d", ErrMalformedEncryption, v)
	}
}

func validatePubKeySecHandler(ctx *model.Context, d types.Dict, pubKeySecHandler bool, subFilter string) error {
	if !pubKeySecHandler {
		return nil
	}

	if subFilter == "adbe.pkcs7.s3" || subFilter == "adbe.pkcs7.s4" {
		obj, ok := d.Find("Recipients")
		if !ok {
			return fmt.Errorf("%w: required entry \"Recipients\" missing", ErrMalformedEncryption)
		}
		arr, err := ctx.DereferenceArray(obj)
		if err != nil {
			return fmt.Errorf("encrypt dict entry \"Recipients\": %w", err)
		}
		if len(arr) == 0 {
			return fmt.Errorf("%w: required entry \"Recipients\" empty", ErrMalformedEncryption)
		}
	}

	return nil
}

func validateEncryptPermissions(
	ctx *model.Context,
	d types.Dict,
	pubKeySecHandler bool,
	subFilter string,
) (int, bool, error, error) {
	p := d.IntEntry("P")
	if p == nil {
		return 0, false, nil, fmt.Errorf("%w: required entry \"P\" missing", ErrMalformedEncryption)
	}

	relaxed := ctx.XRefTable.ValidationMode == model.ValidationRelaxed
	normalizedP, specViolation, err := normalizePermission(*p, relaxed)
	if err != nil {
		return 0, false, nil, err
	}

	encMeta := true
	if emd := d.BooleanEntry("EncryptMetadata"); emd != nil {
		encMeta = *emd
	}

	if err := validatePubKeySecHandler(ctx, d, pubKeySecHandler, subFilter); err != nil {
		return 0, false, nil, err
	}
	return normalizedP, encMeta, specViolation, nil
}

// supportedEncryption returns a pointer to a struct encapsulating used encryption.
func supportedEncryption(ctx *model.Context, d types.Dict) (*model.Enc, error) {
	var specViolations []error

	// Filter
	filter, err := validateEncryptFilter(d)
	if err != nil {
		return nil, err
	}
	pubKeySecHandler := filter == "Adobe.PubSec"

	// SubFilter
	subFilter, err := validateEncryptSubFilter(d, pubKeySecHandler)
	if err != nil {
		return nil, err
	}

	// V
	v, err := validateEncryptV(d)
	if err != nil {
		return nil, err
	}

	// Length
	l, err := validateEncryptLength(d, v)
	if err != nil {
		return nil, err
	}

	// CF, StmF, StrF, EFF
	if v == 4 || v == 5 || v == 6 {
		if err := validateCryptFilters(ctx, d, v, pubKeySecHandler, &specViolations); err != nil {
			return nil, err
		}
	}

	// R
	r, err := getR(ctx, d)
	if err != nil {
		return nil, err
	}

	// O, U
	relaxed := ctx.XRefTable.ValidationMode == model.ValidationRelaxed
	o, u, err := validateOAndU(d, r, relaxed, &specViolations)
	if err != nil {
		return nil, err
	}

	// OE, UE, Perms
	oe, ue, perms, err := validateAES256Parameters(d, r)
	if err != nil {
		return nil, err
	}

	p, encMeta, specViolation, err := validateEncryptPermissions(ctx, d, pubKeySecHandler, subFilter)
	if err != nil {
		return nil, err
	}

	enc := &model.Enc{
		O:     o,
		OE:    oe,
		U:     u,
		UE:    ue,
		L:     l,
		P:     p,
		Perms: perms,
		R:     r,
		V:     v,
		Emd:   encMeta}

	if specViolation != nil {
		appendSpecViolation(&specViolations, specViolation)
	}
	for _, specViolation := range specViolations {
		model.ShowDigestedSpecViolationError(ctx.XRefTable, specViolation)
	}

	return enc, nil
}

func decryptKey(objNumber, generation int, key []byte, aes bool) ([]byte, error) {
	const maxObjectNumber = int64(^uint32(0))

	if objNumber < 0 || int64(objNumber) > maxObjectNumber {
		return nil, fmt.Errorf("decrypt key: object number %d out of range [0,%d]", objNumber, maxObjectNumber)
	}

	if generation < 0 || generation > types.FreeHeadGeneration {
		return nil, fmt.Errorf("decrypt key: generation number %d out of range [0,%d]", generation, types.FreeHeadGeneration)
	}

	m := md5.New()

	nr := uint32(objNumber)
	b1 := []byte{byte(nr), byte(nr >> 8), byte(nr >> 16)}
	b := make([]byte, 0, len(key)+5)
	b = append(b, key...)
	b = append(b, b1...)

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

	return dk, nil
}

// EncryptBytes encrypts s using RC4 or AES.
func encryptBytes(b []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {
	if needAES {
		k := encKey
		if r != 5 && r != 6 {
			var err error
			k, err = decryptKey(objNr, genNr, encKey, needAES)
			if err != nil {
				return nil, err
			}
		}
		return encryptAESBytes(b, k)
	}

	return applyRC4CipherBytes(b, objNr, genNr, encKey, needAES)
}

// decryptBytes decrypts bb using RC4 or AES.
func decryptBytes(b []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {
	if needAES {
		k := encKey
		if r != 5 && r != 6 {
			var err error
			k, err = decryptKey(objNr, genNr, encKey, needAES)
			if err != nil {
				return nil, err
			}
		}
		return decryptAESBytes(b, k)
	}

	return applyRC4CipherBytes(b, objNr, genNr, encKey, needAES)
}

func applyRC4CipherBytes(b []byte, objNr, genNr int, key []byte, needAES bool) ([]byte, error) {
	k, err := decryptKey(objNr, genNr, key, needAES)
	if err != nil {
		return nil, err
	}

	c, err := rc4.NewCipher(k)
	if err != nil {
		return nil, err
	}

	c.XORKeyStream(b, b)

	return b, nil
}

func encrypt(m map[string]types.Object, k string, v types.Object, objNr, genNr int, key []byte, needAES bool, r int) error {
	s, err := encryptDeepObject(v, objNr, genNr, key, needAES, r)
	if err != nil {
		return err
	}

	if s != nil {
		m[k] = s
	}

	return nil
}

func encryptDict(d types.Dict, objNr, genNr int, key []byte, needAES bool, r int) error {
	isSig := false
	ft := d["FT"]
	if ft == nil {
		ft = d["Type"]
	}
	if ft != nil {
		if ftv, ok := ft.(types.Name); ok && (ftv == "Sig" || ftv == "DocTimeStamp") {
			isSig = true
		}
	}
	for k, v := range d {
		if isSig && k == "Contents" {
			continue
		}
		err := encrypt(d, k, v, objNr, genNr, key, needAES, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func encryptStringLiteral(sl types.StringLiteral, objNr, genNr int, key []byte, needAES bool, r int) (*types.StringLiteral, error) {
	bb, err := types.Unescape(sl.Value())
	if err != nil {
		return nil, err
	}

	bb, err = encryptBytes(bb, objNr, genNr, key, needAES, r)
	if err != nil {
		return nil, err
	}

	s, err := types.Escape(string(bb))
	if err != nil {
		return nil, err
	}

	sl = types.StringLiteral(*s)

	return &sl, nil
}

func decryptStringLiteral(sl types.StringLiteral, objNr, genNr int, key []byte, needAES bool, r int) (*types.StringLiteral, error) {
	if sl.Value() == "" {
		return &sl, nil
	}
	bb, err := types.Unescape(sl.Value())
	if err != nil {
		return nil, err
	}

	bb, err = decryptBytes(bb, objNr, genNr, key, needAES, r)
	if err != nil {
		return nil, err
	}

	s, err := types.Escape(string(bb))
	if err != nil {
		return nil, err
	}

	sl = types.StringLiteral(*s)

	return &sl, nil
}

func encryptHexLiteral(hl types.HexLiteral, objNr, genNr int, key []byte, needAES bool, r int) (*types.HexLiteral, error) {
	bb, err := hl.Bytes()
	if err != nil {
		return nil, err
	}

	bb, err = encryptBytes(bb, objNr, genNr, key, needAES, r)
	if err != nil {
		return nil, err
	}

	hl = types.NewHexLiteral(bb)

	return &hl, nil
}

func decryptHexLiteral(hl types.HexLiteral, objNr, genNr int, key []byte, needAES bool, r int) (*types.HexLiteral, error) {
	if hl.Value() == "" {
		return &hl, nil
	}
	bb, err := hl.Bytes()
	if err != nil {
		return nil, err
	}

	bb, err = decryptBytes(bb, objNr, genNr, key, needAES, r)
	if err != nil {
		return nil, err
	}

	hl = types.NewHexLiteral(bb)

	return &hl, nil
}

// EncryptDeepObject recurses over non trivial PDF objects and encrypts all strings encountered.
func encryptDeepObject(objIn types.Object, objNr, genNr int, key []byte, needAES bool, r int) (types.Object, error) {
	_, ok := objIn.(types.IndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case types.StreamDict:
		err := encryptDict(obj.Dict, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}

	case types.Dict:
		err := encryptDict(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}

	case types.Array:
		for i, v := range obj {
			s, err := encryptDeepObject(v, objNr, genNr, key, needAES, r)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = s
			}
		}

	case types.StringLiteral:
		sl, err := encryptStringLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		return *sl, nil

	case types.HexLiteral:
		hl, err := encryptHexLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		return *hl, nil

	default:

	}

	return nil, nil
}

func decryptDict(d types.Dict, objNr, genNr int, key []byte, needAES bool, r int) error {
	isSig := false
	ft := d["FT"]
	if ft == nil {
		ft = d["Type"]
	}
	if ft != nil {
		if ftv, ok := ft.(types.Name); ok && (ftv == "Sig" || ftv == "DocTimeStamp") {
			isSig = true
		}
	}
	for k, v := range d {
		if isSig && k == "Contents" {
			continue
		}
		s, err := decryptDeepObject(v, objNr, genNr, key, needAES, r)
		if err != nil {
			return err
		}
		if s != nil {
			d[k] = s
		}
	}
	return nil
}

func decryptDeepObject(objIn types.Object, objNr, genNr int, key []byte, needAES bool, r int) (types.Object, error) {
	_, ok := objIn.(types.IndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case types.Dict:
		if err := decryptDict(obj, objNr, genNr, key, needAES, r); err != nil {
			return nil, err
		}

	case types.Array:
		for i, v := range obj {
			s, err := decryptDeepObject(v, objNr, genNr, key, needAES, r)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = s
			}
		}

	case types.StringLiteral:
		sl, err := decryptStringLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		return *sl, nil

	case types.HexLiteral:
		hl, err := decryptHexLiteral(obj, objNr, genNr, key, needAES, r)
		if err != nil {
			return nil, err
		}
		return *hl, nil

	default:

	}

	return nil, nil
}

// EncryptStream encrypts a stream buffer using RC4 or AES.
func encryptStream(buf []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {
	k := encKey
	if r != 5 && r != 6 {
		var err error
		k, err = decryptKey(objNr, genNr, encKey, needAES)
		if err != nil {
			return nil, err
		}
	}

	if needAES {
		return encryptAESBytes(buf, k)
	}

	return applyRC4Bytes(buf, k)
}

// decryptStream decrypts a stream buffer using RC4 or AES.
func decryptStream(buf []byte, objNr, genNr int, encKey []byte, needAES bool, r int) ([]byte, error) {
	k := encKey
	if r != 5 && r != 6 {
		var err error
		k, err = decryptKey(objNr, genNr, encKey, needAES)
		if err != nil {
			return nil, err
		}
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
		return nil, errors.New("ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("ciphertext not a multiple of block size")
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
	if len(b) < 2*aes.BlockSize {
		return nil, errAESCiphertextTooShort
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errAESCiphertextUnaligned
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

func fileID(ctx *model.Context) (types.HexLiteral, error) {
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
	if ctx.XRefTable.Version() < model.V20 {
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
	}

	m := h.Sum(nil)

	return types.HexLiteral(strings.ToUpper(hex.EncodeToString(m))), nil
}

func calcFileEncKey(ctx *model.Context) error {
	ctx.EncKey = make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, ctx.EncKey)
	return err
}

func calcOAndUAES256(ctx *model.Context, d types.Dict) (err error) {
	b := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return err
	}

	u := append(make([]byte, 32), b...)
	upw := []byte(ctx.UserPW)
	h := sha256.Sum256(append(upw, validationSalt(u)...))

	ctx.E.U = append(h[:], b...)
	d.Update("U", types.HexLiteral(hex.EncodeToString(ctx.E.U)))

	///////////////////////////////////

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
	d.Update("O", types.HexLiteral(hex.EncodeToString(ctx.E.O)))

	//////////////////////////////////

	if err := calcFileEncKey(ctx); err != nil {
		return err
	}

	//////////////////////////////////

	h = sha256.Sum256(append(upw, keySalt(u)...))
	cb, err := aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	mode := cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.UE, ctx.EncKey)
	d.Update("UE", types.HexLiteral(hex.EncodeToString(ctx.E.UE)))

	//////////////////////////////////

	c = append(opw, keySalt(o)...)
	h = sha256.Sum256(append(c, ctx.E.U...))
	cb, err = aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	mode = cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.OE, ctx.EncKey)
	d.Update("OE", types.HexLiteral(hex.EncodeToString(ctx.E.OE)))

	return nil
}

func calcOAndUAES256Rev6(ctx *model.Context, d types.Dict) (err error) {
	b := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return err
	}

	u := append(make([]byte, 32), b...)
	upw := []byte(ctx.UserPW)
	h, _, err := hashRev6(append(upw, validationSalt(u)...), upw, nil)
	if err != nil {
		return err
	}

	ctx.E.U = append(h[:], b...)
	d.Update("U", types.HexLiteral(hex.EncodeToString(ctx.E.U)))

	///////////////////////////

	b = make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, b)
	if err != nil {
		return err
	}

	o := append(make([]byte, 32), b...)
	opw := []byte(ctx.OwnerPW)
	c := append(opw, validationSalt(o)...)
	h, _, err = hashRev6(append(c, ctx.E.U...), opw, ctx.E.U)
	if err != nil {
		return err
	}

	ctx.E.O = append(h[:], b...)
	d.Update("O", types.HexLiteral(hex.EncodeToString(ctx.E.O)))

	///////////////////////////

	if err := calcFileEncKey(ctx); err != nil {
		return err
	}

	///////////////////////////

	h, _, err = hashRev6(append(upw, keySalt(u)...), upw, nil)
	if err != nil {
		return err
	}

	cb, err := aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	mode := cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.UE, ctx.EncKey)
	d.Update("UE", types.HexLiteral(hex.EncodeToString(ctx.E.UE)))

	//////////////////////////////

	c = append(opw, keySalt(o)...)
	h, _, err = hashRev6(append(c, ctx.E.U...), opw, ctx.E.U)
	if err != nil {
		return err
	}

	cb, err = aes.NewCipher(h[:])
	if err != nil {
		return err
	}

	mode = cipher.NewCBCEncrypter(cb, iv)
	mode.CryptBlocks(ctx.E.OE, ctx.EncKey)
	d.Update("OE", types.HexLiteral(hex.EncodeToString(ctx.E.OE)))

	return nil
}

func calcOAndU(ctx *model.Context, d types.Dict) (err error) {
	if ctx.E.R == 5 {
		return calcOAndUAES256(ctx, d)
	}

	if ctx.E.R == 6 {
		return calcOAndUAES256Rev6(ctx, d)
	}

	ctx.E.O, err = o(ctx)
	if err != nil {
		return err
	}

	ctx.E.U, ctx.EncKey, err = u(ctx)
	if err != nil {
		return err
	}

	d.Update("U", types.HexLiteral(hex.EncodeToString(ctx.E.U)))
	d.Update("O", types.HexLiteral(hex.EncodeToString(ctx.E.O)))

	return nil
}
