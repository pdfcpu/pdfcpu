// Package validate contains validation code for ISO 32000-1:2008.
//
// There is low level validation and validation against the PDF spec for each of the defined PDF object types.
package validate

import (
	"io/ioutil"
	"log"
	"os"
)

var logDebugValidate, logInfoValidate, logErrorValidate *log.Logger

func init() {

	logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorValidate = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {

	if verbose {
		//logDebugValidate = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		//logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

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
