// Package crypto provides PDF encryption plumbing.
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var logDebugCrypto, logInfoCrypto, logErrorCrypto *log.Logger

func init() {
	logDebugCrypto = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugCrypto = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoCrypto = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorCrypto = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	out := ioutil.Discard
	if verbose {
		out = os.Stdout
	}
	logInfoCrypto = log.New(out, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugCrypto = log.New(out, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

var pad = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

func encKey(userpw string, e *types.Enc) (key []byte) {

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

	return
}

// ValidateUserPassword validates userpw.
func ValidateUserPassword(ctx *types.PDFContext) (ok bool, key []byte, err error) {

	// Alg.4/5 p63
	// 4a/5a create enryption key using Alg.2 p61
	fmt.Printf("validateUserPassword: ctx.E = \n%v\n", ctx.E)

	u, key, err := U(ctx)
	if err != nil {
		return
	}

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

	return
}

// O calculates the owner password digest.
func O(ctx *types.PDFContext) ([]byte, error) {

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	fmt.Printf("O: opw=<%s> upw=<%s>\n", ownerpw, userpw)

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
func U(ctx *types.PDFContext) (u []byte, key []byte, err error) {

	userpw := ctx.UserPW
	fmt.Printf("U userpw=%s\n", userpw)

	e := ctx.E

	key = encKey(userpw, e)

	c, err := rc4.NewCipher(key)
	if err != nil {
		return
	}

	if e.R == 2 {
		// 4b
		u = make([]byte, 32)
		copy(u, pad)
		c.XORKeyStream(u, u)
	} else {

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
				return
			}
			c.XORKeyStream(u, u)
		}
	}

	return u, key, nil
}

// ValidateOwnerPassword validates ownerpw.
func ValidateOwnerPassword(ctx *types.PDFContext) (ok bool, err error) {

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	fmt.Printf("ValidateOwnerPassword: opw=%s upw=%s\n", ownerpw, userpw)

	e := ctx.E

	// 7a: Alg.3 p62 a-d
	key := key(ownerpw, userpw, e.R, e.L)

	// 7b
	upw := make([]byte, len(e.O))
	copy(upw, e.O)

	switch e.R {

	case 2:
		c, err := rc4.NewCipher(key)
		if err != nil {
			return false, err
		}
		c.XORKeyStream(upw, upw)

	case 3, 4:
		for i := 19; i >= 0; i-- {

			keynew := make([]byte, len(key))
			copy(keynew, key)

			for j := range keynew {
				keynew[j] ^= byte(i)
			}

			c, err := rc4.NewCipher(keynew)
			if err != nil {
				return false, err
			}

			c.XORKeyStream(upw, upw)
		}
	}

	upws := ctx.UserPW
	ctx.UserPW = string(upw)
	ok, _, err = ValidateUserPassword(ctx)
	if err != nil {
		return false, err
	}
	ctx.UserPW = upws

	return ok, nil
}

// SupportedCFEntry returns true if all entries found entries are supported.
func SupportedCFEntry(d *types.PDFDict) (bool, bool) {

	cfm := d.NameEntry("CFM")
	if cfm != nil && *cfm != "V2" && *cfm != "AESV2" {
		logErrorCrypto.Println("supportedCFEntry: invalid entry \"CFM\"")
		return false, false
	}

	ae := d.NameEntry("AuthEvent")
	if ae != nil && *ae != "DocOpen" {
		logErrorCrypto.Println("supportedCFEntry: invalid entry \"AuthEvent\"")
		return false, false
	}

	l := d.IntEntry("Length")
	if l != nil && (*l < 8 || *l > 128 || *l%8 > 1) {
		logErrorCrypto.Println("supportedCFEntry: invalid entry \"Length\"")
		return false, false
	}

	return cfm != nil && *cfm == "AESV2", true
}

func printP(enc *types.Enc) {

	p := enc.P

	bits := "4,5"
	if enc.R >= 3 {
		bits = "10,11"
	}

	logInfoCrypto.Printf("permission to process needs bits: %s\n", bits)
	logDebugCrypto.Printf("P: %d -> %0b\n", p, uint32(p)&0x0F3C)
	logInfoCrypto.Printf("Bit  3: %t (print)\n", p&0x0004 > 0)
	logInfoCrypto.Printf("Bit  4: %t (modify)\n", p&0x0008 > 0)
	logInfoCrypto.Printf("Bit  5: %t (copy, extract)\n", p&0x0010 > 0)
	logInfoCrypto.Printf("Bit  6: %t (add or modify annotations)\n", p&0x0020 > 0)
	logInfoCrypto.Printf("Bit  9: %t (fill in form fields)\n", p&0x0100 > 0)
	logInfoCrypto.Printf("Bit 10: %t (extract)\n", p&0x0200 > 0)
	logInfoCrypto.Printf("Bit 11: %t (assemble)\n", p&0x0400 > 0)
	logInfoCrypto.Printf("Bit 12: %t (print high-level)\n", p&0x0800 > 0)
}

// HasNeededPermissions returns true if permissions for pdfcpu processing are present.
func HasNeededPermissions(enc *types.Enc) bool {

	//return true
	// see 7.6.3.2

	printP(enc)

	if enc.R >= 3 {
		// needs set bits 10 and 11
		return enc.P&0x0200 > 0 && enc.P&0x0400 > 0
	}

	// R == 2
	// needs set bits 4 and 5
	return enc.P&0x0008 > 0 && enc.P&0x0010 > 0
}

// SupportedEncryption returns true if used encryption is supported by pdfcpu.
func SupportedEncryption(ctx *types.PDFContext, dict *types.PDFDict) (*types.Enc, error) {

	var aes, ok bool
	var err error

	// Filter
	filter := dict.NameEntry("Filter")
	if filter == nil || *filter != "Standard" {
		logErrorCrypto.Println("supportedEncryption: Filter must be \"Standard\"")
		return nil, nil
	}

	// SubFilter
	if dict.NameEntry("SubFilter") != nil {
		logErrorCrypto.Println("supportedEncryption: \"SubFilter\" not supported")
		return nil, nil
	}

	// V
	v := dict.IntEntry("V")
	if v == nil || (*v != 1 && *v != 2 && *v != 4) {
		logErrorCrypto.Println("supportedEncryption: \"V\" must be one of 1,2,4")
		return nil, nil
	}

	// Length
	length := 40
	l := dict.IntEntry("Length")
	if l != nil {
		if *l < 40 || *l > 128 || *l%8 > 0 {
			logErrorCrypto.Printf("supportedEncryption: \"Length\" %d not supported\n", *l)
			return nil, nil
		}
		length = *l
	}

	if *v == 4 {

		// CF
		cfDict := dict.PDFDictEntry("CF")
		if cfDict == nil {
			logErrorCrypto.Println("supportedEncryption: required entry \"CF\" missing.")
			return nil, nil
		}

		// StmF
		stmf := dict.NameEntry("StmF")
		if stmf != nil && *stmf != "Identity" {
			d := cfDict.PDFDictEntry(*stmf)
			if d == nil {
				logErrorCrypto.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *stmf)
				return nil, nil
			}
			aes, ok = SupportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *stmf)
			}
			ctx.AES4Streams = aes
		}

		// StrF
		strf := dict.NameEntry("StrF")
		if strf != nil && *strf != "Identity" {
			d := cfDict.PDFDictEntry(*strf)
			if d == nil {
				logErrorCrypto.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *strf)
				return nil, nil
			}
			aes, ok = SupportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *strf)
			}
			ctx.AES4Strings = aes
		}

		// EFF
		eff := dict.NameEntry("EFF")
		if eff != nil && *strf != "Identity" {
			d := cfDict.PDFDictEntry(*eff)
			if d == nil {
				logErrorCrypto.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *eff)
				return nil, nil
			}
			aes, ok = SupportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *eff)
			}
			ctx.AES4EmbeddedStreams = aes
		}

	}

	// R
	r := dict.IntEntry("R")
	if r == nil || (*r != 2 && *r != 3 && *r != 4) {
		logErrorCrypto.Println("supportedEncryption: \"R\" must be 2,3,4")
		return nil, nil
	}

	// O
	o, err := dict.StringEntryBytes("O")
	if err != nil {
		return nil, err
	}
	if o == nil || len(o) != 32 {
		logErrorCrypto.Println("supportedEncryption: required entry \"O\" missing or invalid")
		return nil, nil
	}

	// U
	u, err := dict.StringEntryBytes("U")
	if err != nil {
		return nil, err
	}
	if u == nil || len(u) != 32 {
		logErrorCrypto.Printf("supportedEncryption: required entry \"U\" missing or invalid %d", len(u))
		return nil, nil
	}

	// P
	p := dict.IntEntry("P")
	if p == nil {
		logErrorCrypto.Println("supportedEncryption: required entry \"P\" missing")
		return nil, nil
	}

	// EncryptMetadata
	encMeta := true
	emd := dict.BooleanEntry("EncryptMetadata")
	if emd != nil {
		encMeta = *emd
	}

	return &types.Enc{O: o, U: u, L: length, P: *p, R: *r, V: *v, Emd: encMeta}, nil
}

func decryptKey(objNumber, generation int, key []byte, aes bool) []byte {

	logDebugCrypto.Printf("decryptKey: obj:%d gen:%d key:%x aes:%t\n", objNumber, generation, key, aes)

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

	logDebugCrypto.Printf("decryptKey returning: %X\n", dk)

	return dk
}

// EncryptString encrypts s using RC4 or AES.
func EncryptString(needAES bool, s string, objNr, genNr int, key []byte) (*string, error) {

	logInfoCrypto.Printf("EncryptString begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	var s1 *string
	var err error
	k := decryptKey(objNr, genNr, key, needAES)

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

	return types.Escape(*s1)
}

// DecryptString decrypts s using RC4 or AES.
func DecryptString(needAES bool, s string, objNr, genNr int, key []byte) (*string, error) {

	logInfoCrypto.Printf("DecryptString begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	b, err := types.Unescape(s)
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

	logDebugCrypto.Printf("applyRC4Cipher begin s:<%v> %d %d key:%X aes:%t\n", b, objNr, genNr, key, needAES)

	c, err := rc4.NewCipher(decryptKey(objNr, genNr, key, needAES))
	if err != nil {
		return nil, err
	}

	c.XORKeyStream(b, b)
	s1 := string(b)
	logDebugCrypto.Printf("applyRC4Cipher end, rc4 returning: <%s>\n", s1)

	return &s1, nil
}

// EncryptDeepObject recurses over non trivial PDF objects and encrypts all strings encountered.
func EncryptDeepObject(objIn interface{}, objNr, genNr int, key []byte, aes bool) (*types.PDFStringLiteral, error) {

	_, ok := objIn.(types.PDFIndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case types.PDFDict:
		for k, v := range obj.Dict {
			s, err := EncryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj.Dict[k] = *s
			}
		}

	case types.PDFArray:
		for i, v := range obj {
			s, err := EncryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case types.PDFStringLiteral:
		s, err := EncryptString(aes, obj.Value(), objNr, genNr, key)
		if err != nil {
			return nil, err
		}

		sl := types.PDFStringLiteral(*s)

		return &sl, nil

	default:

	}

	return nil, nil
}

// DecryptDeepObject recurses over non trivial PDF objects and decrypts all strings encountered.
func DecryptDeepObject(objIn interface{}, objNr, genNr int, key []byte, aes bool) (*types.PDFStringLiteral, error) {

	_, ok := objIn.(types.PDFIndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case types.PDFDict:
		for k, v := range obj.Dict {
			s, err := DecryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj.Dict[k] = *s
			}
		}

	case types.PDFArray:
		for i, v := range obj {
			s, err := DecryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case types.PDFStringLiteral:
		s, err := DecryptString(aes, obj.Value(), objNr, genNr, key)
		if err != nil {
			return nil, err
		}

		sl := types.PDFStringLiteral(*s)

		return &sl, nil

	default:

	}

	return nil, nil
}

// EncryptStream encrypts a stream buffer using RC4 or AES.
func EncryptStream(needAES bool, buf []byte, objNr, genNr int, key []byte) ([]byte, error) {

	logInfoCrypto.Printf("EncryptStream begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	k := decryptKey(objNr, genNr, key, needAES)

	if needAES {
		return encryptAESBytes(buf, k)
	}

	return applyRC4Bytes(buf, k)

}

// DecryptStream decrypts a stream buffer using RC4 or AES.
func DecryptStream(needAES bool, buf []byte, objNr, genNr int, key []byte) ([]byte, error) {

	logInfoCrypto.Printf("DecryptStream begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

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

	return data, nil
}

func decryptAESBytes(b, key []byte) (data []byte, err error) {

	if len(b) < aes.BlockSize {
		return nil, errors.New("decryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("decryptAESBytes: Ciphertext not a multiple of block size")
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	iv := make([]byte, aes.BlockSize)
	copy(iv, b[:aes.BlockSize])

	data = b[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(data, data)

	// Remove padding.
	// Note: For some reason not all AES ciphertexts are padded.
	if len(data) > 0 && data[len(data)-1] <= 0x10 {
		e := len(data) - int(data[len(data)-1])
		data = data[:e]
	}

	return
}
