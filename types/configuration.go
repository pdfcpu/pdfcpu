package types

const (

	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict = 0

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed = 1

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"

	// PermissionsAll enables all user access permission bits.
	PermissionsAll int16 = -1 // 0xFFFF

	// PermissionsNone disables all user access permissions bits.
	PermissionsNone int16 = -3901 // 0xF0C3

)

// CommandMode specifies the operation being executed.
type CommandMode int

// The available commands.
const (
	VALIDATE CommandMode = iota
	OPTIMIZE
	SPLIT
	MERGE
	EXTRACTIMAGES
	EXTRACTFONTS
	EXTRACTPAGES
	EXTRACTCONTENT
	TRIM
	ADDATTACHMENTS
	REMOVEATTACHMENTS
	EXTRACTATTACHMENTS
	LISTATTACHMENTS
	ADDPERMISSIONS
	LISTPERMISSIONS
	ENCRYPT
	DECRYPT
	CHANGEUPW
	CHANGEOPW
)

// Configuration of a PDFContext.
type Configuration struct {

	// Enables PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enables decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed
	ValidationMode int

	// End of line char sequence for writing.
	Eol string

	// Turns on object stream generation.
	// A signal for compressing any new non-stream-object into an object stream.
	// true enforces WriteXRefStream to true.
	// false does not prevent xRefStream generation.
	WriteObjectStream bool

	// Switches between xRefSection (<=V1.4) and objectStream/xRefStream (>=V1.5) writing.
	WriteXRefStream bool

	// Turns on stats collection.
	CollectStats bool

	// A CSV-filename holding the statistics.
	StatsFileName string

	// Supplied user password
	UserPW    string
	UserPWNew *string

	// Supplied owner password
	OwnerPW    string
	OwnerPWNew *string

	// EncryptUsingAES ensures AES encryption.
	// true: AES encryption
	// false: RC4 encryption.
	EncryptUsingAES bool

	// EncryptUsing128BitKey ensures 128 bit key length.
	// true: use 128 bit key
	// false: use 40 bit key
	EncryptUsing128BitKey bool

	// Supplied user access permissions, see Table 22
	UserAccessPermissions int16

	// Command being executed.
	Mode CommandMode
}

// NewDefaultConfiguration returns the default pdfcpu configuration.
func NewDefaultConfiguration() *Configuration {

	return &Configuration{
		Reader15:              true,
		DecodeAllStreams:      false,
		ValidationMode:        ValidationRelaxed,
		Eol:                   EolLF,
		WriteObjectStream:     true,
		WriteXRefStream:       true,
		CollectStats:          true,
		EncryptUsingAES:       true,
		EncryptUsing128BitKey: true,
		UserAccessPermissions: PermissionsNone,
	}
}

// ValidationModeString returns a string rep for the validation mode in effect.
func (c *Configuration) ValidationModeString() string {

	if c.ValidationMode == ValidationStrict {
		return "strict"
	}

	if c.ValidationMode == ValidationRelaxed {
		return "relaxed"
	}

	return ""
}

// SetValidationStrict sets strict validation.
func (c *Configuration) SetValidationStrict() {
	c.ValidationMode = ValidationStrict
}

// SetValidationRelaxed sets relaxed validation.
func (c *Configuration) SetValidationRelaxed() {
	c.ValidationMode = ValidationRelaxed
}
