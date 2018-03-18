// Package validate contains validation code for ISO 32000-1:2008.
//
// There is low level validation and validation against the PDF spec for each of the defined PDF object types.
package validate

func memberOf(s string, list []string) bool {

	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}

func intMemberOf(i int, list []int) bool {
	for _, v := range list {
		if i == v {
			return true
		}
	}
	return false
}
