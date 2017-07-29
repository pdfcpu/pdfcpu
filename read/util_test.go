package read

import "testing"

func TestPositionToNextWhitespaceOrChar(t *testing.T) {

	str := "12345"
	positionToNextWhitespaceOrChar(str, "")
	//t.Logf("<%s> positioned:<%s> at pos:%d\n", str, newstr, i)

	str = "before   after"
	positionToNextWhitespaceOrChar(str, "")
	//t.Logf("<%s> positioned:<%s> at pos:%d\n", str, newstr, i)

	str = "   text"
	positionToNextWhitespaceOrChar(str, "")
	//t.Logf("<%s> positioned:<%s> at pos:%d\n", str, newstr, i)

	str = "text [abc"
	positionToNextWhitespaceOrChar(str, "[<")
	//t.Logf("<%s> positioned:<%s> at pos:%d\n", str, newstr, i)

	str = "text[ abc"
	positionToNextWhitespaceOrChar(str, "[<")
	//t.Logf("<%s> positioned:<%s> at pos:%d\n", str, newstr, i)
}

func TestTrimLeftSpace(t *testing.T) {

	str := "   %     x"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "   %\x0Aabc"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "   %\x0Dabc"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "   %\x0D%     \x0A\x0Ddef"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "        x"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "x"
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "   x       "
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = ""
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "         "
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)

	str = "   \t    "
	trimLeftSpace(str)
	//t.Logf("raw<%s> trimmedLeftSpace<%s> number of trimmed ws:%d\n", str, s, wsCount)
}

func TestHexString(t *testing.T) {

	str := "0c"
	_, ok := hexString(str)
	if ok {
		//t.Logf("%s isHexString:%v hexstr=%s\n", str, ok, *hexstr)
	} else {
		t.Errorf("%s isHexString:%v\n", str, ok)
	}

	str = "0CFE"
	_, ok = hexString(str)
	if ok {
		//t.Logf("%s isHexString:%v hexstr=%s\n", str, ok, *hexstr)
	} else {
		t.Errorf("%s isHexString:%v\n", str, ok)
	}

	str = "0BF"
	_, ok = hexString(str)
	if ok {
		//t.Logf("%s isHexString:%v hexstr=%s\n", str, ok, *hexstr)
	} else {
		t.Errorf("%s isHexString:%v\n", str, ok)
	}

	str = "0BGF"
	hexstr, ok := hexString(str)
	if ok {
		t.Errorf("%s isHexString:%v hexstr=%s\n", str, ok, *hexstr)
	} else {
		//t.Logf("%s isHexString:%v\n", str, ok)
	}
}

func doTestParseBalancedParenthesesOK(str string, t *testing.T) {
	if i := balancedParenthesesPrefix(str); i >= 0 {
		//t.Logf("%s isLiteralString: i=%d str=%s\n", str, i, str[:i+1])
	} else {
		t.Error("balancedParenthesesPrefix failed")
	}
}

func doTestParseBalancedParenthesesFail(str string, t *testing.T) {
	if i := balancedParenthesesPrefix(str); i >= 0 {
		t.Error("balancedParenthesesPrefix: should have failed")
	}
}

func TestBalancedParentheses(t *testing.T) {

	// Positive testing
	doTestParseBalancedParenthesesOK("()", t)
	doTestParseBalancedParenthesesOK("(a\\na\\ra\\ta\\baa\\675a\\faaaaa\\\x28aaa\\\x29aaaaaaa)", t)
	doTestParseBalancedParenthesesOK("(a\x5c()", t)
	doTestParseBalancedParenthesesOK("(a\\a())", t)
	doTestParseBalancedParenthesesOK("((a)(b)(345\\(#$%#)^([]/<>))", t)

	// Negative testing
	doTestParseBalancedParenthesesFail("(", t)
	doTestParseBalancedParenthesesFail("(akl()", t)
	doTestParseBalancedParenthesesFail("((a)(<<)", t)
}

func doTestParseString(str string, t *testing.T) {

	//outstr := stringLiteral(str)
	//t.Logf("str:\n%s\n% x\noutstr:\n%s\n% x\n", str, str, outstr, outstr)
}

func TestParseString(t *testing.T) {

	// split lines by \eol, 3 variations
	doTestParseString("(strings may contain newlines \\\x0aand such.)", t)
	doTestParseString("(strings may contain newlines \\\x0dand such.)", t)
	doTestParseString("(strings may contain newlines \\\x0d\x0aand such.)", t)

	//	// inline eol, 3 variations
	doTestParseString("(strings may contain newlines\x0aand such.)", t)
	doTestParseString("(strings may contain newlines\x0dand such.)", t)
	doTestParseString("(strings may contain newlines\x0a\x0aand such.)", t)

	// defined escape sequences
	doTestParseString("(allowed escape sequences: \\n \\r \\t \\b \\f \\( \\) \\\\)", t)

	// skip undefined escape sequence
	doTestParseString("(undefined escape sequences: \\c\\d\\e)", t)

	// octal code escape sequences
	doTestParseString("(\\100X\\000\\101)", t)
	doTestParseString("(\\1\\2\\3)", t)
	doTestParseString("(\\01\\02\\03)", t)
	doTestParseString("(\\001\\002\\003)", t)
	doTestParseString("(\\100X\\01\\6X\\100)", t)
	doTestParseString("(\\78)", t)
	doTestParseString("(octal values: \\755 \\13)", t)

	doTestParseString("(Strings may contain balanced parentheses ( ) \\\x0a"+
		"and special characters (*!&}^% and so on).)", t)

	doTestParseString("(\\b\\12a\\\x0a      b  %c   \x0d      c)", t)

	doTestParseString("(This string has an end-of-line at the end of it.\\\x0d)", t)
	doTestParseString("(so does this one.\\n)", t)

	doTestParseString("(\\0053)", t)

	doTestParseString("()", t)

}

func TestGetInt(t *testing.T) {

	i := getInt([]byte{})
	if i != 0 {
		t.Errorf("getInt returning:%d\n", i)
	}

	i = getInt([]byte(""))
	if i != 0 {
		t.Errorf("getInt returning:%d\n", i)
	}

	i = getInt([]byte("\x80"))
	if i != 128 {
		t.Errorf("getInt returning:%d\n", i)
	}

	i = getInt([]byte("\x01\x00"))
	if i != 256 {
		t.Errorf("getInt returning:%d\n", i)
	}

	i = getInt([]byte("\x01\x00\x00"))
	if i != 65536 {
		t.Errorf("getInt returning:%d\n", i)
	}
}

func TestHexStringToUTF16String(t *testing.T) {

	hexStr := "FEFF004F00700065006E004F00660066006900630065002E006F0072006700200033002E0031"

	_, err := decodeUTF16String(hexStr)
	if err != nil {
		t.Errorf("decodeUTF16String failed %v", err)
	} else {
		//t.Logf("hex:%s utf16:%s\n", hexStr, s)
	}

	hexStr = "FFFE004F00700065006E004F00660066006900630065002E006F0072006700200033002E0031"
	_, err = decodeUTF16String(hexStr)
	if err != nil {
		//t.Errorf("decodeUTF16String failed %v", err)
	} else {
		t.Errorf("decodeUTF16String should have failed: %s\n", hexStr)
	}
}
