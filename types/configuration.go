package types

const (

	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict = 0

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed = 1

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"
)

// Configuration of a PDFContext.
type Configuration struct {

	// Enables PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enables decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed
	ValidationMode int

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
}

// NewDefaultConfiguration returns the default pdflib configuration.
func NewDefaultConfiguration() *Configuration {
	return &Configuration{
		Reader15:          true,
		DecodeAllStreams:  false,
		ValidationMode:    ValidationRelaxed,
		WriteObjectStream: true,
		WriteXRefStream:   true,
		CollectStats:      true,
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
