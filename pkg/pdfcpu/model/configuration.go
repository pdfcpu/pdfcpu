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

package model

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	// ConfigurationSchemaVersionLegacy identifies configurations without explicit schema metadata.
	ConfigurationSchemaVersionLegacy = 0

	// ConfigurationSchemaVersionCurrent identifies the newest configuration schema supported by this build.
	ConfigurationSchemaVersionCurrent = 1
)

var (
	// ErrInvalidConfigurationSchema signals malformed or unsupported configuration schema metadata.
	ErrInvalidConfigurationSchema = errors.New("invalid configuration schema")
)

var schema1ConfigurationKeys = map[string]struct{}{
	"allowedRevocationHosts":          {},
	"checkFileNameExt":                {},
	"createBookmarks":                 {},
	"created":                         {},
	"dateFormat":                      {},
	"decodeAllStreams":                {},
	"encryptKeyLength":                {},
	"encryptUsingAES":                 {},
	"eol":                             {},
	"formFieldListMaxColWidth":        {},
	"maxDecodeBytes":                  {},
	"maxImageBytes":                   {},
	"maxImagePixels":                  {},
	"maxInputBytes":                   {},
	"maxObjectBytes":                  {},
	"maxStreamBytes":                  {},
	"needAppearances":                 {},
	"offline":                         {},
	"optimize":                        {},
	"optimizeBeforeWriting":           {},
	"optimizeDuplicateContentStreams": {},
	"optimizeResourceDicts":           {},
	"permissions":                     {},
	"postProcessValidate":             {},
	"preferredCertRevocationChecker":  {},
	"reader15":                        {},
	"schemaVersion":                   {},
	"timeout":                         {},
	"timeoutCRL":                      {},
	"timeoutOCSP":                     {},
	"timestampFormat":                 {},
	"unit":                            {},
	"validationMode":                  {},
	"writeObjectStream":               {},
	"writeXRefStream":                 {},
}

func schema1ConfigurationKeySet(keys []string) (map[string]bool, error) {
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			return nil, fmt.Errorf("invalid schema 1 configuration: duplicate key %q", key)
		}
		if _, ok := schema1ConfigurationKeys[key]; !ok {
			return nil, fmt.Errorf("invalid schema 1 configuration: unknown key %q", key)
		}
		seen[key] = true
	}
	return seen, nil
}

const (
	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict int = iota

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed
)

// PermissionFlags represents the user access permissions defined by PDF 32000 table 22.
type PermissionFlags int

// User access permission bits.
const (
	UnusedFlag1              PermissionFlags = 1 << iota // Bit 1:  unused
	UnusedFlag2                                          // Bit 2:  unused
	PermissionPrintRev2                                  // Bit 3:  Print (security handlers rev.2), draft print (security handlers >= rev.3)
	PermissionModify                                     // Bit 4:  Modify contents by operations other than controlled by bits 6, 9, 11.
	PermissionExtract                                    // Bit 5:  Copy, extract text & graphics
	PermissionModAnnFillForm                             // Bit 6:  Add or modify annotations, fill form fields, in conjunction with bit 4 create/mod form fields.
	UnusedFlag7                                          // Bit 7:  unused
	UnusedFlag8                                          // Bit 8:  unused
	PermissionFillRev3                                   // Bit 9:  Fill form fields (security handlers >= rev.3)
	PermissionExtractRev3                                // Bit 10: Copy, extract text & graphics (security handlers >= rev.3) (unused since PDF 2.0)
	PermissionAssembleRev3                               // Bit 11: Assemble document (security handlers >= rev.3)
	PermissionPrintRev3                                  // Bit 12: Print (security handlers >= rev.3)
)

// Common permission sets.
const (
	PermissionsNone  = PermissionFlags(0xF0C3)
	PermissionsPrint = PermissionsNone + PermissionPrintRev2 + PermissionPrintRev3
	PermissionsAll   = PermissionFlags(0xFFFF)
)

// MergeBookmarkMode controls how merge creates or preserves bookmarks.
type MergeBookmarkMode string

const (
	// MergeBookmarkModeWrap creates one top-level bookmark per merged file.
	MergeBookmarkModeWrap MergeBookmarkMode = "wrap"

	// MergeBookmarkModePreserve merges existing bookmark roots without per-file wrapper bookmarks.
	MergeBookmarkModePreserve MergeBookmarkMode = "preserve"
)

const (

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"
)

// CommandMode specifies the operation being executed.
type CommandMode int

// The available commands.
const (
	VALIDATE CommandMode = iota
	LISTINFO
	OPTIMIZE
	SPLIT
	SPLITBYPAGENR
	MERGECREATE
	MERGECREATEZIP
	MERGEAPPEND
	EXTRACTIMAGES
	EXTRACTFONTS
	EXTRACTPAGES
	EXTRACTCONTENT
	EXTRACTMETADATA
	TRIM
	LISTATTACHMENTS
	EXTRACTATTACHMENTS
	ADDATTACHMENTS
	ADDATTACHMENTSPORTFOLIO
	REMOVEATTACHMENTS
	LISTPERMISSIONS
	SETPERMISSIONS
	ADDWATERMARKS
	REMOVEWATERMARKS
	IMPORTIMAGES
	INSERTPAGESBEFORE
	INSERTPAGESAFTER
	REMOVEPAGES
	LISTKEYWORDS
	ADDKEYWORDS
	REMOVEKEYWORDS
	LISTPROPERTIES
	ADDPROPERTIES
	REMOVEPROPERTIES
	COLLECT
	CROP
	LISTBOXES
	ADDBOXES
	REMOVEBOXES
	LISTANNOTATIONS
	ADDANNOTATIONS
	REMOVEANNOTATIONS
	ROTATE
	NUP
	GRID
	BOOKLET
	LISTBOOKMARKS
	ADDBOOKMARKS
	REMOVEBOOKMARKS
	IMPORTBOOKMARKS
	EXPORTBOOKMARKS
	LISTIMAGES
	UPDATEIMAGES
	CREATE
	DUMP
	LISTFORMFIELDS
	REMOVEFORMFIELDS
	LOCKFORMFIELDS
	UNLOCKFORMFIELDS
	RESETFORMFIELDS
	EXPORTFORMFIELDS
	FILLFORMFIELDS
	MULTIFILLFORMFIELDS
	ENCRYPT
	DECRYPT
	CHANGEUPW
	CHANGEOPW
	CHEATSHEETSFONTS
	INSTALLFONTS
	LISTFONTS
	RESIZE
	POSTER
	NDOWN
	CUT
	LISTPAGELAYOUT
	SETPAGELAYOUT
	RESETPAGELAYOUT
	LISTPAGEMODE
	SETPAGEMODE
	RESETPAGEMODE
	LISTVIEWERPREFERENCES
	SETVIEWERPREFERENCES
	RESETVIEWERPREFERENCES
	ZOOM
	LISTCERTIFICATES
	INSPECTCERTIFICATES
	IMPORTCERTIFICATES
	VALIDATESIGNATURES
	REMOVESIGNATURES
	ADDSIGNATURE
)

// AllowRemoveEncryption enables removing encryption during validation.
func (cmd CommandMode) AllowRemoveEncryption() bool {
	return cmd == OPTIMIZE || cmd == REMOVESIGNATURES
}

// AllowRemoveSignatures enables removing signatures during validation.
func (cmd CommandMode) AllowRemoveSignatures() bool {
	return cmd == MERGEAPPEND || cmd == MERGECREATE || cmd == MERGECREATEZIP || cmd == OPTIMIZE
}

type configurationResourceMode uint8

const (
	configurationResourceModeAuto configurationResourceMode = iota
	configurationResourceModeAutoIsolated
	configurationResourceModeReadOnly
	configurationResourceModeStateless
)

type configurationResources struct {
	mode           configurationResourceMode
	userFontDir    string
	trustedCertDir string
}

func resourcesForConfigurationDir(mode configurationResourceMode, dir string) configurationResources {
	return configurationResources{
		mode:           mode,
		userFontDir:    filepath.Join(dir, "fonts"),
		trustedCertDir: filepath.Join(dir, "certs"),
	}
}

// Configuration of a Context.
type Configuration struct {
	resources configurationResources

	// Location of corresponding config.yml
	Path string

	CreationDate string

	// Version records legacy generator metadata and does not determine configuration compatibility.
	// Deprecated: configuration compatibility is defined by SchemaVersion.
	Version string

	// SchemaVersion identifies the configuration file schema independently of the pdfcpu product version.
	SchemaVersion int

	// Ensure .pdf input file extension.
	CheckFileNameExt bool

	// Enable PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enable decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed.
	ValidationMode int

	// UnsupportedResourcePolicy controls unsupported-resource handling during extraction.
	// This is a runtime option and is not read from config.yml.
	UnsupportedResourcePolicy UnsupportedResourcePolicy

	// Enable validation right before writing.
	PostProcessValidate bool

	// PreserveInfoDict preserves existing Producer, CreationDate and ModDate entry objects after input validation.
	// It does not bypass validation, restore discarded Info data or preserve complete serialized Info dictionary bytes.
	// This is a runtime option and is not read from config.yml.
	PreserveInfoDict bool

	// Check for broken links in LinkedAnnotations/URIActions.
	ValidateLinks bool

	// End of line char sequence for writing.
	Eol string

	// Turn on object stream generation.
	// A signal for compressing any new non-stream-object into an object stream.
	// true enforces WriteXRefStream to true.
	// false does not prevent xRefStream generation.
	WriteObjectStream bool

	// Switch between xRefSection (<=V1.4) and objectStream/xRefStream (>=V1.5) writing.
	WriteXRefStream bool

	// CSV filename holding input file statistics.
	StatsFileName string

	// Supplied user password.
	UserPW    string
	UserPWNew *string

	// Supplied owner password.
	OwnerPW    string
	OwnerPWNew *string

	// Supplied private key password.
	PrivateKeyPW string

	// EncryptUsingAES ensures AES encryption.
	// true: AES encryption
	// false: RC4 encryption.
	EncryptUsingAES bool

	// AES:40,128,256 RC4:40,128
	EncryptKeyLength int

	// Supplied user access permissions, see Table 22.
	Permissions PermissionFlags // int16

	// Command being executed.
	Cmd CommandMode

	// Display unit in effect.
	Unit types.DisplayUnit

	// Timestamp format.
	TimestampFormat string

	// Date format.
	DateFormat string

	// Optimize after reading and validating the xreftable but before processing.
	Optimize bool

	// Optimize after processing but before writing.
	// TODO add to config.yml
	OptimizeBeforeWriting bool

	// Optimize page resources via content stream analysis. (assuming Optimize == true || OptimizeBeforeWriting == true)
	OptimizeResourceDicts bool

	// Optimize duplicate content streams across pages. (assuming Optimize == true || OptimizeBeforeWriting == true)
	OptimizeDuplicateContentStreams bool

	// Merge creates bookmarks.
	CreateBookmarks bool

	// MergeBookmarkMode controls how merge creates or preserves bookmarks.
	// This is a runtime option and is not read from config.yml.
	MergeBookmarkMode MergeBookmarkMode

	// PDF Viewer is expected to supply appearance streams for form fields.
	NeedAppearances bool

	// Internet availability.
	Offline bool

	// HTTP timeout in seconds.
	Timeout int

	// Http timeout in seconds for CRL revocation checking.
	TimeoutCRL int

	// Http timeout in seconds for OCSP revocation checking.
	TimeoutOCSP int

	// AllowedRevocationHosts contains private hosts explicitly trusted for CRL and OCSP requests.
	AllowedRevocationHosts []string

	// Preferred certificate revocation checking mechanism: CRL, OSCP
	PreferredCertRevocationChecker int

	// Limit form field content for display purposes when using pdfcpu form list.
	// If > 0 affects the columns AltName, Default and Value.
	FormFieldListMaxColWidth int

	// Limits controls resource usage for input-driven allocation.
	Limits ResourceLimits

	// Do not encrypt output files.
	RemoveEncryption bool

	// Remove existing signatures.
	RemoveSignatures bool
}

// Clone returns an independent copy of c and preserves nil receiver semantics.
func (c *Configuration) Clone() *Configuration {
	if c == nil {
		return nil
	}

	clone := *c
	clone.AllowedRevocationHosts = slices.Clone(c.AllowedRevocationHosts)
	if c.UserPWNew != nil {
		userPWNew := *c.UserPWNew
		clone.UserPWNew = &userPWNew
	}
	if c.OwnerPWNew != nil {
		ownerPWNew := *c.OwnerPWNew
		clone.OwnerPWNew = &ownerPWNew
	}
	return &clone
}

// TrustedCertificateStore returns the certificate directory selected for c.
// available is false when certificate resources are intentionally unavailable.
func (c *Configuration) TrustedCertificateStore() (dir string, available bool) {
	if c != nil {
		switch c.resources.mode {
		case configurationResourceModeAutoIsolated:
			if c.resources.trustedCertDir != "" {
				return c.resources.trustedCertDir, true
			}
		case configurationResourceModeReadOnly:
			return c.resources.trustedCertDir, true
		case configurationResourceModeStateless:
			return "", false
		}
	}
	return TrustedCertDir, true
}

// UserFontStore returns the user-font directory selected for c.
// available is false when user-font resources are intentionally unavailable.
func (c *Configuration) UserFontStore() (dir string, available bool) {
	if c != nil {
		switch c.resources.mode {
		case configurationResourceModeAutoIsolated:
			if c.resources.userFontDir != "" {
				return c.resources.userFontDir, true
			}
		case configurationResourceModeReadOnly:
			return c.resources.userFontDir, true
		case configurationResourceModeStateless:
			return "", false
		}
	}
	return font.UserFontDir, true
}

// ResourceLimits controls resource usage for input-driven allocation.
type ResourceLimits struct {
	// MaxInputBytes limits each PDF input and stdin spool. Zero means unlimited.
	MaxInputBytes int64

	// MaxObjectBytes limits an indirect-object buffer, excluding stream payloads.
	// Zero selects the default of 64 MiB.
	MaxObjectBytes int64

	// MaxStreamBytes limits encoded stream bytes read from a PDF.
	MaxStreamBytes int64

	// MaxDecodeBytes limits decoded stream bytes produced by filters.
	MaxDecodeBytes int64

	// MaxImagePixels limits decoded/rendered image dimensions.
	MaxImagePixels int64

	// MaxImageBytes limits decoded/rendered image buffer sizes.
	MaxImageBytes int64

	// MaxObjectCount limits xref stream /Size expansion.
	MaxObjectCount int

	// MaxObjectStreamCount limits object stream /N.
	MaxObjectStreamCount int

	// MaxObjectStreamFirst limits object stream /First prolog bytes.
	MaxObjectStreamFirst int64

	// MaxXRefEntries limits xref stream Index expansion.
	MaxXRefEntries int

	// MaxRecursionDepth limits recursive parsing and object graph traversal.
	MaxRecursionDepth int
}

// DefaultResourceLimits returns the default resource limits.
func DefaultResourceLimits() ResourceLimits {
	const (
		MB = 1 << 20
		MP = 1_000_000
	)

	return ResourceLimits{
		MaxObjectBytes:       64 * MB,
		MaxStreamBytes:       512 * MB,
		MaxDecodeBytes:       512 * MB,
		MaxImagePixels:       100 * MP,
		MaxImageBytes:        512 * MB,
		MaxObjectCount:       10_000_000,
		MaxObjectStreamCount: 1_000_000,
		MaxObjectStreamFirst: 16 * MB,
		MaxXRefEntries:       10_000_000,
		MaxRecursionDepth:    100,
	}
}

// ConfigPath defines the location of pdfcpu's configuration directory.
// If set to a file path, pdfcpu will ensure the config dir at this location.
// Other possible values:
//
//	default:	Ensure config dir at default location
//	disable:	Disable config dir usage
//
// If you want to disable config dir usage in a multi threaded environment
// you are encouraged to use api.DisableConfigDir().
var ConfigPath string = "default"

var loadedDefaultConfig *Configuration

//go:embed resources/config.yml
var configFileBytes []byte

//go:embed resources/Roboto-Regular.ttf
var robotoFontFileBytes []byte

func defaultConfigurationFileBytes() []byte {
	header := fmt.Sprintf(`
#############################
#   Default configuration   #
#############################

# Creation date
created: %s 

`,
		time.Now().Format("2006-01-02 15:04"))

	return append([]byte(header), configFileBytes...)
}

func initializeConfigurationFile(path string) error {
	return fileutil.WriteFile(path, defaultConfigurationFileBytes(), 0600)
}

func readOpenConfigurationFile(f *os.File, path string) (conf *Configuration, err error) {
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	return readConfiguration(f, path)
}

func readConfigurationFile(path string) (*Configuration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return readOpenConfigurationFile(f, path)
}

func readConfigurationAt(root string) (*Configuration, error) {
	if root == "" {
		return nil, errors.New("missing configuration root")
	}
	configDir := filepath.Join(root, "pdfcpu")
	path := filepath.Join(configDir, "config.yml")
	if _, err := preflightConfigurationSchema(path, ConfigurationSchemaVersionCurrent); err != nil {
		return nil, err
	}
	conf, err := readConfigurationFile(path)
	if err != nil {
		return nil, err
	}
	conf.resources = resourcesForConfigurationDir(configurationResourceModeReadOnly, configDir)
	return conf, nil
}

// LoadConfigurationReadOnly loads an existing configuration tree rooted at root without modifying filesystem or package
// loader state.
func LoadConfigurationReadOnly(root string) (*Configuration, error) {
	return readConfigurationAt(root)
}

func loadOrInitializeConfigurationFile(path string) (*Configuration, bool, error) {
	if _, err := preflightConfigurationSchema(path, ConfigurationSchemaVersionCurrent); err == nil {
		conf, err := readConfigurationFile(path)
		return conf, false, err
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	if err := initializeConfigurationFile(path); err != nil {
		return nil, false, err
	}
	conf, err := readConfigurationFile(path)
	return conf, true, err
}

func ensureConfigFileAt(path string, override bool) error {
	if !override {
		f, err := os.Open(path)
		if err == nil {
			conf, err := readOpenConfigurationFile(f, path)
			if err != nil {
				return err
			}
			conf.resources = resourcesForConfigurationDir(configurationResourceModeAuto, filepath.Dir(path))
			loadedDefaultConfig = conf
			return nil
		}
	}

	if err := initializeConfigurationFile(path); err != nil {
		return err
	}
	conf, err := readConfigurationFile(path)
	if err != nil {
		return err
	}
	conf.resources = resourcesForConfigurationDir(configurationResourceModeAuto, filepath.Dir(path))
	loadedDefaultConfig = conf
	return nil
}

func onlyHidden(files []os.DirEntry) bool {
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return false
		}
	}
	return true
}

// ensureFontDirInitializedAt sets up the font directory without loading fonts.
// Font loading is deferred until fonts are actually needed.
func ensureFontDirInitializedAt(userFontDir string) error {
	files, err := os.ReadDir(userFontDir)
	if err != nil {
		return err
	}
	if onlyHidden(files) {
		// Ensure Roboto font for form filling.
		fontname := "Roboto-Regular"
		if log.DebugEnabled() && log.CLIEnabled() {
			log.CLI.Printf("installing user font: %s\n", fontname)
		}
		if err := font.InstallFontFromBytesQuiet(userFontDir, fontname, robotoFontFileBytes); err != nil {
			return err
		}
	}

	return nil
}

func ensureFontDirInitialized() error {
	return ensureFontDirInitializedAt(font.UserFontDir)
}

func initCertificatesAt(trustedCertDir string) error {
	files, err := os.ReadDir(trustedCertDir)
	if err != nil {
		return err
	}
	if !onlyHidden(files) {
		return nil
	}
	if !bundledDefaultCertificates {
		return nil
	}

	MarkCertificateStoreChanged()
	return installDefaultCertificates(trustedCertDir)
}

func initCertificates() error {
	return initCertificatesAt(TrustedCertDir)
}

func initializeConfigurationResourcesAt(configDir string) error {
	userFontDir := filepath.Join(configDir, "fonts")
	if err := os.MkdirAll(userFontDir, 0755); err != nil {
		return err
	}
	if err := ensureFontDirInitializedAt(userFontDir); err != nil {
		return err
	}

	trustedCertDir := filepath.Join(configDir, "certs")
	if err := os.MkdirAll(trustedCertDir, 0755); err != nil {
		return err
	}
	return initCertificatesAt(trustedCertDir)
}

func loadConfiguration(root string) (*Configuration, bool, error) {
	if root == "" {
		return nil, false, errors.New("missing configuration root")
	}

	configDir := filepath.Join(root, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, false, err
	}
	conf, created, err := loadOrInitializeConfigurationFile(filepath.Join(configDir, "config.yml"))
	if err != nil {
		return nil, false, err
	}
	conf.resources = resourcesForConfigurationDir(configurationResourceModeAutoIsolated, configDir)
	if err := initializeConfigurationResourcesAt(configDir); err != nil {
		return nil, false, err
	}
	return conf, created, nil
}

// InitializeConfiguration loads or initializes a configuration tree and reports whether config.yml was created.
func InitializeConfiguration(root string) (*Configuration, bool, error) {
	return loadConfiguration(root)
}

// LoadConfiguration loads or initializes a configuration tree rooted at root without changing compatibility loader
// state.
func LoadConfiguration(root string) (*Configuration, error) {
	conf, _, err := loadConfiguration(root)
	return conf, err
}

// ResetConfiguration replaces config.yml with the built-in configuration at root without changing compatibility loader
// state. Existing font and certificate resources are preserved.
func ResetConfiguration(root string) (*Configuration, error) {
	if root == "" {
		return nil, errors.New("missing configuration root")
	}

	configDir := filepath.Join(root, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(configDir, "config.yml")
	if err := initializeConfigurationFile(path); err != nil {
		return nil, err
	}
	conf, err := readConfigurationFile(path)
	if err != nil {
		return nil, err
	}
	conf.resources = resourcesForConfigurationDir(configurationResourceModeAutoIsolated, configDir)
	if err := initializeConfigurationResourcesAt(configDir); err != nil {
		return nil, err
	}
	return conf, nil
}

// EnsureDefaultConfigAt tries to load the default configuration from path.
// If path/pdfcpu/config.yaml is not found, it will be created.
func EnsureDefaultConfigAt(path string, override bool) error {
	configDir := filepath.Join(path, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := ensureConfigFileAt(filepath.Join(configDir, "config.yml"), override); err != nil {
		return err
	}

	// Initialize pdfcpu config/fonts dir for userfonts then extract and install Roboto as default Unicode font for form
	// filling.
	// Other userfonts have to be installed via `pdfcpu font install` or copied over from another pdfcpu config dir.
	// Userfonts are loaded into memory lazily.
	font.UserFontDir = filepath.Join(configDir, "fonts")
	if err := os.MkdirAll(font.UserFontDir, 0755); err != nil {
		return err
	}
	if err := ensureFontDirInitialized(); err != nil {
		return err
	}

	// Initialize pdfcpu config/cert dir, then extract and install certificates.
	// Certificates are loaded into memory lazily.
	TrustedCertDir = filepath.Join(configDir, "certs")
	if err := os.MkdirAll(TrustedCertDir, 0755); err != nil {
		return err
	}
	if err := initCertificates(); err != nil {
		return err
	}

	//fmt.Println(loadedDefaultConfig)

	return nil
}

// NewStatelessConfiguration returns an independent configuration using built-in defaults without accessing filesystem
// state.
func NewStatelessConfiguration() *Configuration {
	// NOTE: Needs to stay in sync with config.yml
	//
	// Takes effect whenever the installed config.yml is disabled:
	// 		cli: supply -conf disable
	// 		api: call api.DisableConfigDir()
	return &Configuration{
		resources: configurationResources{mode: configurationResourceModeStateless},

		CreationDate:                    time.Now().Format("2006-01-02 15:04"),
		SchemaVersion:                   ConfigurationSchemaVersionCurrent,
		CheckFileNameExt:                true,
		Reader15:                        true,
		DecodeAllStreams:                false,
		ValidationMode:                  ValidationRelaxed,
		UnsupportedResourcePolicy:       UnsupportedResourceSkip,
		ValidateLinks:                   false,
		Eol:                             types.EolLF,
		WriteObjectStream:               true,
		WriteXRefStream:                 true,
		EncryptUsingAES:                 true,
		EncryptKeyLength:                256,
		Permissions:                     PermissionsPrint,
		TimestampFormat:                 "2006-01-02 15:04",
		DateFormat:                      "2006-01-02",
		Optimize:                        true,
		OptimizeBeforeWriting:           true,
		OptimizeResourceDicts:           true,
		OptimizeDuplicateContentStreams: false,
		CreateBookmarks:                 true,
		MergeBookmarkMode:               MergeBookmarkModeWrap,
		NeedAppearances:                 false,
		Offline:                         false,
		Timeout:                         5,
		TimeoutCRL:                      10,
		TimeoutOCSP:                     10,
		PreferredCertRevocationChecker:  CRL,
		FormFieldListMaxColWidth:        0,
		Limits:                          DefaultResourceLimits(),
	}
}

func validateConfigurationSchemaVersion(version int) error {
	if err := validateConfigurationSchemaVersionValue(version); err != nil {
		return err
	}
	if version > ConfigurationSchemaVersionCurrent {
		return fmt.Errorf(
			"%w: schemaVersion %d exceeds supported version %d",
			ErrInvalidConfigurationSchema,
			version,
			ConfigurationSchemaVersionCurrent,
		)
	}
	return nil
}

func validateConfigurationSchemaVersionValue(version int) error {
	if version <= ConfigurationSchemaVersionLegacy {
		return fmt.Errorf("%w: schemaVersion must be a positive integer, got %d", ErrInvalidConfigurationSchema, version)
	}
	return nil
}

// ResetConfig resets the default configuration.
func ResetConfig() error {
	path, err := os.UserConfigDir()
	if err != nil {
		path = os.TempDir()
	}
	return EnsureDefaultConfigAt(path, true)
}

// NewDefaultConfiguration returns the default pdfcpu configuration.
func NewDefaultConfiguration() *Configuration {
	if loadedDefaultConfig != nil {
		c := *loadedDefaultConfig
		c.AllowedRevocationHosts = slices.Clone(loadedDefaultConfig.AllowedRevocationHosts)
		return &c
	}
	if ConfigPath != "disable" {
		path, err := os.UserConfigDir()
		if err != nil {
			path = os.TempDir()
		}
		if err = EnsureDefaultConfigAt(path, false); err == nil {
			c := *loadedDefaultConfig
			c.AllowedRevocationHosts = slices.Clone(loadedDefaultConfig.AllowedRevocationHosts)
			return &c
		}
		fault.Fail("config problem: %w", err)
	}
	// Bypass config.yml
	return NewStatelessConfiguration()
}

// NewAESConfiguration returns a default configuration for AES encryption.
func NewAESConfiguration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = true
	c.EncryptKeyLength = keyLength
	return c
}

// NewRC4Configuration returns a default configuration for RC4 encryption.
func NewRC4Configuration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = false
	c.EncryptKeyLength = keyLength
	return c
}

// EolString returns a string rep for the eol in effect.
func (c *Configuration) EolString() string {
	var s string
	switch c.Eol {
	case types.EolLF:
		s = "EolLF"
	case types.EolCR:
		s = "EolCR"
	case types.EolCRLF:
		s = "EolCRLF"
	}
	return s
}

// ValidationModeString returns a string rep for the validation mode in effect.
func (c *Configuration) ValidationModeString() string {
	if c.ValidationMode == ValidationStrict {
		return "strict"
	}
	return "relaxed"
}

// PreferredCertRevocationCheckerString returns a string rep for the preferred certificate revocation checker in effect.
func (c *Configuration) PreferredCertRevocationCheckerString() string {
	if c.PreferredCertRevocationChecker == CRL {
		return "CRL"
	}
	return "OSCP"
}

// UnitString returns a string rep for the display unit in effect.
func (c *Configuration) UnitString() string {
	var s string
	switch c.Unit {
	case types.POINTS:
		s = "points"
	case types.INCHES:
		s = "inches"
	case types.CENTIMETRES:
		s = "cm"
	case types.MILLIMETRES:
		s = "mm"
	}
	return s
}

// SetUnit configures the display unit.
func (c *Configuration) SetUnit(s string) {
	switch s {
	case "points":
		c.Unit = types.POINTS
	case "inches":
		c.Unit = types.INCHES
	case "cm":
		c.Unit = types.CENTIMETRES
	case "mm":
		c.Unit = types.MILLIMETRES
	}
}

// ApplyReducedFeatureSet returns true if complex entries like annotations shall not be written.
func (c *Configuration) ApplyReducedFeatureSet() bool {
	switch c.Cmd {
	case SPLIT, TRIM, EXTRACTPAGES, IMPORTIMAGES:
		return true
	}
	return false
}
