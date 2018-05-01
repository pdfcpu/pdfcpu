package pdfcpu

// Functions dealing with PDF encryption.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

var (
	pad = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}

	nullPad = []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Needed permission bits for pdfcpu commands.
	perm = map[CommandMode]struct{ extract, modify int }{
		VALIDATE:           {0, 0},
		OPTIMIZE:           {0, 0},
		SPLIT:              {1, 0},
		MERGE:              {0, 0},
		EXTRACTIMAGES:      {1, 0},
		EXTRACTFONTS:       {1, 0},
		EXTRACTPAGES:       {1, 0},
		EXTRACTCONTENT:     {1, 0},
		TRIM:               {0, 1},
		LISTATTACHMENTS:    {0, 0},
		EXTRACTATTACHMENTS: {1, 0},
		ADDATTACHMENTS:     {0, 1},
		REMOVEATTACHMENTS:  {0, 1},
		LISTPERMISSIONS:    {0, 0},
		ADDPERMISSIONS:     {0, 0},
	}
)

// NewEncryptDict creates a new EncryptDict using the standard security handler.
func newEncryptDict(needAES, need128BitKey bool, permissions int16) *PDFDict {

	d := NewPDFDict()

	//d.Insert("Type", PDFName("Encrypt"))

	d.Insert("Filter", PDFName("Standard"))

	if need128BitKey {
		d.Insert("Length", PDFInteger(128))
		d.Insert("R", PDFInteger(4))
		d.Insert("V", PDFInteger(4))
	} else {
		d.Insert("R", PDFInteger(2))
		d.Insert("V", PDFInteger(1))
	}

	// Set user access permission flags.
	d.Insert("P", PDFInteger(permissions))

	d.Insert("StmF", PDFName("StdCF"))
	d.Insert("StrF", PDFName("StdCF"))

	d1 := NewPDFDict()
	d1.Insert("AuthEvent", PDFName("DocOpen"))

	if needAES {
		d1.Insert("CFM", PDFName("AESV2"))
	} else {
		d1.Insert("CFM", PDFName("V2"))
	}

	if need128BitKey {
		d1.Insert("Length", PDFInteger(16))
	} else {
		d1.Insert("Length", PDFInteger(5))
	}

	d2 := NewPDFDict()
	d2.Insert("StdCF", d1)

	d.Insert("CF", d2)

	h := "0000000000000000000000000000000000000000000000000000000000000000"
	d.Insert("U", PDFHexLiteral(h))
	d.Insert("O", PDFHexLiteral(h))

	return &d
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

// ValidateUserPassword validates the user password aka document open password.
func validateUserPassword(ctx *PDFContext) (ok bool, key []byte, err error) {

	// Alg.4/5 p63
	// 4a/5a create encryption key using Alg.2 p61

	//fmt.Printf("validateUserPassword: ctx.E.U =\n%v\n", ctx.E.U)

	u, key, err := u(ctx)
	if err != nil {
		return false, nil, err
	}

	//fmt.Printf("validateUserPassword: u =\n%v\n", u)

	return bytes.HasPrefix(ctx.E.U, u), key, nil
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
func o(ctx *PDFContext) ([]byte, error) {

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	//fmt.Printf("O: opw=<%s> upw=<%s>\n", ownerpw, userpw)

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
func u(ctx *PDFContext) (u []byte, key []byte, err error) {

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
		u = append(u, nullPad[:32-len(u)]...)
	}

	return u, key, nil
}

// ValidateOwnerPassword validates the owner password aka change permissions password.
func validateOwnerPassword(ctx *PDFContext) (ok bool, k []byte, err error) {

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	//fmt.Printf("ValidateOwnerPassword: ownerpw=ctx.OwnerPW=%s userpw=ctx.UserPW=%s\n", ownerpw, userpw)

	e := ctx.E

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
				return false, nil, err
			}

			c.XORKeyStream(upw, upw)
		}
	}

	// Save user pw
	upws := ctx.UserPW

	ctx.UserPW = string(upw)
	ok, k, err = validateUserPassword(ctx)

	// Restore user pw
	ctx.UserPW = upws

	return ok, k, err
}

// SupportedCFEntry returns true if all entries found are supported.
func supportedCFEntry(d *PDFDict) (bool, error) {

	cfm := d.NameEntry("CFM")
	if cfm != nil && *cfm != "V2" && *cfm != "AESV2" {
		return false, errors.New("supportedCFEntry: invalid entry \"CFM\"")
	}

	ae := d.NameEntry("AuthEvent")
	if ae != nil && *ae != "DocOpen" {
		return false, errors.New("supportedCFEntry: invalid entry \"AuthEvent\"")
	}

	l := d.IntEntry("Length")
	if l != nil && (*l < 8 || *l > 128 || *l%8 > 1) {
		return false, errors.New("supportedCFEntry: invalid entry \"Length\"")
	}

	return cfm != nil && *cfm == "AESV2", nil
}

func perms(p int) (list []string) {

	list = append(list, fmt.Sprintf("%0b", uint32(p)&0x0F3C))
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

func permissions(ctx *PDFContext) (list []string) {

	if ctx.E == nil {
		return append(list, "full access")
	}

	return perms(ctx.E.P)
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

func getV(dict *PDFDict) (*int, error) {

	v := dict.IntEntry("V")

	if v == nil || (*v != 1 && *v != 2 && *v != 4) {
		return nil, errors.Errorf("getV: \"V\" must be one of 1,2,4")
	}

	return v, nil
}
func checkStmf(ctx *PDFContext, stmf *string, cfDict *PDFDict) error {

	if stmf != nil && *stmf != "Identity" {

		d := cfDict.PDFDictEntry(*stmf)
		if d == nil {
			return errors.Errorf("checkStmf: entry \"%s\" missing in \"CF\"", *stmf)
		}

		aes, err := supportedCFEntry(d)
		if err != nil {
			return errors.Wrapf(err, "checkStmv: unsupported \"%s\" entry in \"CF\"", *stmf)
		}
		ctx.AES4Streams = aes
	}

	return nil
}

func checkV(ctx *PDFContext, dict *PDFDict) (*int, error) {

	v, err := getV(dict)
	if err != nil {
		return nil, err
	}

	// Right now we support only 4
	if *v != 4 {
		return v, nil
	}

	// CF
	cfDict := dict.PDFDictEntry("CF")
	if cfDict == nil {
		return nil, errors.Errorf("checkV: required entry \"CF\" missing.")
	}

	// StmF
	stmf := dict.NameEntry("StmF")
	err = checkStmf(ctx, stmf, cfDict)
	if err != nil {
		return nil, err
	}

	// StrF
	strf := dict.NameEntry("StrF")
	if strf != nil && *strf != "Identity" {
		d := cfDict.PDFDictEntry(*strf)
		if d == nil {
			return nil, errors.Errorf("checkV: entry \"%s\" missing in \"CF\"", *strf)
		}
		aes, err := supportedCFEntry(d)
		if err != nil {
			return nil, errors.Wrapf(err, "checkV: unsupported \"%s\" entry in \"CF\"", *strf)
		}
		ctx.AES4Strings = aes
	}

	// EFF
	eff := dict.NameEntry("EFF")
	if eff != nil && *strf != "Identity" {
		d := cfDict.PDFDictEntry(*eff)
		if d == nil {
			return nil, errors.Errorf("checkV: entry \"%s\" missing in \"CF\"", *eff)
		}
		aes, err := supportedCFEntry(d)
		if err != nil {
			return nil, errors.Wrapf(err, "checkV: unsupported \"%s\" entry in \"CF\"", *strf)
		}
		ctx.AES4EmbeddedStreams = aes
	}

	return v, nil
}

func length(dict *PDFDict) (int, error) {

	l := dict.IntEntry("Length")
	if l == nil {
		return 40, nil
	}

	if *l < 40 || *l > 128 || *l%8 > 0 {
		return 0, errors.Errorf("length: \"Length\" %d not supported\n", *l)
	}

	return *l, nil
}

func getR(dict *PDFDict) (int, error) {

	r := dict.IntEntry("R")
	if r == nil || (*r != 2 && *r != 3 && *r != 4) {
		return 0, errors.New("getR: \"R\" must be 2,3,4")
	}

	return *r, nil
}

// SupportedEncryption returns true if used encryption is supported by pdfcpu
// Also returns a pointer to a struct encapsulating used encryption.
func supportedEncryption(ctx *PDFContext, dict *PDFDict) (*Enc, error) {

	// Filter
	filter := dict.NameEntry("Filter")
	if filter == nil || *filter != "Standard" {
		return nil, errors.New("unsupported encryption: filter must be \"Standard\"")
	}

	// SubFilter
	if dict.NameEntry("SubFilter") != nil {
		return nil, errors.New("unsupported encryption: \"SubFilter\" not supported")
	}

	// V
	v, err := checkV(ctx, dict)
	if err != nil {
		return nil, err
	}

	// Length
	l, err := length(dict)
	if err != nil {
		return nil, err
	}

	// R
	r, err := getR(dict)
	if err != nil {
		return nil, err
	}

	// O
	o, err := dict.StringEntryBytes("O")
	if err != nil {
		return nil, err
	}
	if o == nil || len(o) != 32 {
		return nil, errors.New("unsupported encryption: required entry \"O\" missing or invalid")
	}

	// U
	u, err := dict.StringEntryBytes("U")
	if err != nil {
		return nil, err
	}
	if u == nil || len(u) != 32 {
		return nil, errors.Errorf("unsupported encryption: required entry \"U\" missing or invalid %d", len(u))
	}

	// P
	p := dict.IntEntry("P")
	if p == nil {
		return nil, errors.New("unsupported encryption: required entry \"P\" missing")
	}

	// EncryptMetadata
	encMeta := true
	emd := dict.BooleanEntry("EncryptMetadata")
	if emd != nil {
		encMeta = *emd
	}

	return &Enc{O: o, U: u, L: l, P: *p, R: r, V: *v, Emd: encMeta}, nil
}

func decryptKey(objNumber, generation int, key []byte, aes bool) []byte {

	log.Debug.Printf("decryptKey: obj:%d gen:%d key:%x aes:%t\n", objNumber, generation, key, aes)

	m := md5.New()

	nr := uint32(objNumber)
	b1 := []byte{byte(nr), byte(nr >> 8), byte(nr >> 16)}
	b := append(key, b1...)

	gen := uint16(generation)
	b2 := []byte{byte(gen), byte(gen >> 8)}
	b = append(b, b2...)

	//logDebugCrypto.Printf("b: %X\n", b)

	m.Write(b)

	if aes {
		m.Write([]byte("sAlT"))
	}

	dk := m.Sum(nil)

	l := len(key) + 5
	if l < 16 {
		dk = dk[:l]
	}

	log.Debug.Printf("decryptKey returning: %X\n", dk)

	return dk
}

// EncryptString encrypts s using RC4 or AES.
func encryptString(needAES bool, s string, objNr, genNr int, key []byte) (*string, error) {

	log.Debug.Printf("EncryptString begin obj:%d gen:%d key:%X aes:%t\n<%s>\n", objNr, genNr, key, needAES, s)

	var s1 *string
	var err error
	k := decryptKey(objNr, genNr, key, needAES)
	//logInfoCrypto.Printf("EncryptString k = %v\n", k)

	if needAES {
		b, err := encryptAESBytes([]byte(s), k)
		if err != nil {
			return nil, err
		}
		sb := string(b)
		s1 = &sb

	} else {
		s1, err = applyRC4Cipher([]byte(s), objNr, genNr, key, needAES)
		if err != nil {
			return nil, err
		}
	}

	return Escape(*s1)
}

// DecryptString decrypts s using RC4 or AES.
func decryptString(needAES bool, s string, objNr, genNr int, key []byte) (*string, error) {

	log.Debug.Printf("DecryptString begin obj:%d gen:%d key:%X aes:%t s:<%s>\n", objNr, genNr, key, needAES, s)

	b, err := Unescape(s)
	if err != nil {
		return nil, err
	}

	k := decryptKey(objNr, genNr, key, needAES)

	if needAES {
		b, err = decryptAESBytes(b, k)
		if err != nil {
			return nil, err
		}
		s1 := string(b)
		return &s1, nil
	}

	return applyRC4Cipher(b, objNr, genNr, key, needAES)
}

func applyRC4Cipher(b []byte, objNr, genNr int, key []byte, needAES bool) (*string, error) {

	log.Debug.Printf("applyRC4Cipher begin s:<%v> %d %d key:%X aes:%t\n", b, objNr, genNr, key, needAES)

	c, err := rc4.NewCipher(decryptKey(objNr, genNr, key, needAES))
	if err != nil {
		return nil, err
	}

	c.XORKeyStream(b, b)
	s1 := string(b)
	log.Debug.Printf("applyRC4Cipher end, rc4 returning: <%s>\n", s1)

	return &s1, nil
}

func encrypt(m map[string]PDFObject, k string, v PDFObject, objNr, genNr int, key []byte, aes bool) error {

	s, err := encryptDeepObject(v, objNr, genNr, key, aes)
	if err != nil {
		return err
	}

	if s != nil {
		m[k] = *s
	}

	return nil
}

// EncryptDeepObject recurses over non trivial PDF objects and encrypts all strings encountered.
func encryptDeepObject(objIn PDFObject, objNr, genNr int, key []byte, aes bool) (*PDFStringLiteral, error) {

	_, ok := objIn.(PDFIndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case PDFStreamDict:
		for k, v := range obj.Dict {
			err := encrypt(obj.Dict, k, v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
		}

	case PDFDict:
		for k, v := range obj.Dict {
			err := encrypt(obj.Dict, k, v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
		}

	case PDFArray:
		for i, v := range obj {
			s, err := encryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case PDFStringLiteral:
		s, err := encryptString(aes, obj.Value(), objNr, genNr, key)
		if err != nil {
			return nil, err
		}

		sl := PDFStringLiteral(*s)

		return &sl, nil

	default:

	}

	return nil, nil
}

// DecryptDeepObject recurses over non trivial PDF objects and decrypts all strings encountered.
func decryptDeepObject(objIn PDFObject, objNr, genNr int, key []byte, aes bool) (*PDFStringLiteral, error) {

	_, ok := objIn.(PDFIndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case PDFDict:
		for k, v := range obj.Dict {
			s, err := decryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj.Dict[k] = *s
			}
		}

	case PDFArray:
		for i, v := range obj {
			s, err := decryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case PDFStringLiteral:
		s, err := decryptString(aes, obj.Value(), objNr, genNr, key)
		if err != nil {
			return nil, err
		}

		sl := PDFStringLiteral(*s)

		return &sl, nil

	default:

	}

	return nil, nil
}

// EncryptStream encrypts a stream buffer using RC4 or AES.
func encryptStream(needAES bool, buf []byte, objNr, genNr int, key []byte) ([]byte, error) {

	log.Debug.Printf("EncryptStream begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	k := decryptKey(objNr, genNr, key, needAES)

	if needAES {
		return encryptAESBytes(buf, k)
	}

	return applyRC4Bytes(buf, k)
}

// DecryptStream decrypts a stream buffer using RC4 or AES.
func decryptStream(needAES bool, buf []byte, objNr, genNr int, key []byte) ([]byte, error) {

	log.Debug.Printf("DecryptStream begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	k := decryptKey(objNr, genNr, key, needAES)

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

	//fmt.Printf("encryptAESBytes before:\n%s\n", hex.Dump(b))

	// pad b to aes.Blocksize
	l := len(b) % aes.BlockSize
	c := 0x10
	if l > 0 {
		c = aes.BlockSize - l
	}
	b = append(b, bytes.Repeat([]byte{byte(c)}, aes.BlockSize-l)...)

	if len(b) < aes.BlockSize {
		return nil, errors.New("encryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("encryptAESBytes: Ciphertext not a multiple of block size")
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

	//fmt.Printf("encryptAESBytes after:\n%s\n", hex.Dump(data))

	return data, nil
}

func decryptAESBytes(b, key []byte) ([]byte, error) {

	//fmt.Printf("decryptAESBytes before:\n%s\n", hex.Dump(b))

	if len(b) < aes.BlockSize {
		return nil, errors.New("decryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("decryptAESBytes: Ciphertext not a multiple of block size")
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

	//fmt.Printf("decryptAESBytes after:\n%s\n", hex.Dump(data))

	return data, nil
}

func fileID(ctx *PDFContext) PDFHexLiteral {

	// see also 14.4 File Identifiers.

	h := md5.New()
	h.Write([]byte(time.Now().String())) // current timestamp.
	//h.Write() file location - ignore, we don't have this.
	h.Write([]byte(strconv.Itoa(int(ctx.Read.FileSize)))) // file size.
	// h.Write(allValuesOfTheInfoDict) - ignore, does not make sense in this case because we patch the info dict.
	m := h.Sum(nil)

	return PDFHexLiteral(hex.EncodeToString(m))
}

// ID generates the ID element for this file.
func id(ctx *PDFContext) *PDFArray {

	// Generate a PDFArray for the ID element.

	fid := fileID(ctx)
	return &PDFArray{fid, fid}
}
