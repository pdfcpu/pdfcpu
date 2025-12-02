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

package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

// Command-specific options structs
type validateOptions struct {
	mode     string
	links    bool
	optimize bool
}

type encryptOptions struct {
	mode string
	key  string
	perm string
}

type extractOptions struct {
	mode string
}

type splitOptions struct {
	mode string
}

type mergeOptions struct {
	mode         string
	bookmarks    bool
	dividerPage  bool
	optimize     bool
	sorted       bool
	bookmarksSet bool
	optimizeSet  bool
}

type watermarkOptions struct {
	mode string
}

type stampOptions struct {
	mode string
}

type formMultifillOptions struct {
	mode string
}

type pagesInsertOptions struct {
	mode string
}

type optimizeCommandOptions struct {
	fileStats string
}

type bookmarksImportOptions struct {
	replaceBookmarks bool
}

type infoOptions struct {
	fonts bool
	json  bool
}

type signaturesValidateOptions struct {
	all  bool
	full bool
}

type viewerpreferencesListOptions struct {
	all  bool
	json bool
}

type annotationsRemoveOptions struct {
	all bool
}

type boxesRemoveOptions struct {
	all bool
}

type certificatesListOptions struct {
	json bool
}

func init() {
	// Register all commands
	rootCmd.AddCommand(annotationsCmd())
	rootCmd.AddCommand(attachmentsCmd())
	rootCmd.AddCommand(bookmarksCmd())
	rootCmd.AddCommand(bookletCmd())
	rootCmd.AddCommand(boxesCmd())
	rootCmd.AddCommand(certificatesCmd())
	rootCmd.AddCommand(changeopwCmd())
	rootCmd.AddCommand(changeupwCmd())
	rootCmd.AddCommand(collectCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(cropCmd())
	rootCmd.AddCommand(cutCmd())
	rootCmd.AddCommand(decryptCmd())
	rootCmd.AddCommand(encryptCmd())
	rootCmd.AddCommand(extractCmd())
	rootCmd.AddCommand(fontsCmd())
	rootCmd.AddCommand(formCmd())
	rootCmd.AddCommand(gridCmd())
	rootCmd.AddCommand(imagesCmd())
	rootCmd.AddCommand(importCmd())
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(keywordsCmd())
	rootCmd.AddCommand(mergeCmd())
	rootCmd.AddCommand(ndownCmd())
	rootCmd.AddCommand(nupCmd())
	rootCmd.AddCommand(optimizeCmd())
	rootCmd.AddCommand(pagelayoutCmd())
	rootCmd.AddCommand(pagemodeCmd())
	rootCmd.AddCommand(pagesCmd())
	rootCmd.AddCommand(paperCmd())
	rootCmd.AddCommand(permissionsCmd())
	rootCmd.AddCommand(portfolioCmd())
	rootCmd.AddCommand(posterCmd())
	rootCmd.AddCommand(propertiesCmd())
	rootCmd.AddCommand(resizeCmd())
	rootCmd.AddCommand(rotateCmd())
	rootCmd.AddCommand(selectedpagesCmd())
	rootCmd.AddCommand(signaturesCmd())
	rootCmd.AddCommand(splitCmd())
	rootCmd.AddCommand(stampCmd())
	rootCmd.AddCommand(trimCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(viewerprefCmd())
	rootCmd.AddCommand(watermarkCmd())
	rootCmd.AddCommand(zoomCmd())
	rootCmd.AddCommand(completionCmd())
}

// Helper function to wrap handlers
func wrapHandler(handler func(*model.Configuration, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		conf := getConfig()
		if conf.Version != model.VersionStr {
			model.CheckConfigVersion(conf.Version)
		}
		handler(conf, args)
	}
}

func annotationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotations",
		Short: "List, remove page annotations",
		Long:  usageLongAnnots,
	}

	list := &cobra.Command{
		Use:   "list inFile",
		Short: "List annotations",
		Args:  cobra.MinimumNArgs(1),
		Run:   wrapHandler(processListAnnotationsCommand),
	}
	list.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	remove := &cobra.Command{
		Use:   "remove inFile [outFile] [objNr|annotId|annotType]...",
		Short: "Remove annotations",
		Args:  cobra.MinimumNArgs(1),
		Run:   wrapHandler(processRemoveAnnotationsCommand),
	}
	remove.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	cmd.AddCommand(list, remove)
	return cmd
}

func attachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "List, add, remove, extract embedded file attachments",
		Long:  usageLongAttach,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List attachments",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "add inFile file...",
			Short: "Add attachments",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processAddAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [file...]",
			Short: "Remove attachments",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processRemoveAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "extract inFile outDir [file...]",
			Short: "Extract attachments",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processExtractAttachmentsCommand),
		},
	)

	return cmd
}

func bookmarksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "List, import, export, remove bookmarks",
		Long:  usageLongBookmarks,
	}

	importOpts := &bookmarksImportOptions{replaceBookmarks: false}
	importCmd := &cobra.Command{
		Use:   "import inFile inFileJSON [outFile]",
		Short: "Import bookmarks",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processImportBookmarksCommand(conf, args, importOpts)
		},
	}
	importCmd.Flags().BoolVarP(&importOpts.replaceBookmarks, "replace", "r", importOpts.replaceBookmarks, "replace existing bookmarks")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List bookmarks",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListBookmarksCommand),
		},
		importCmd,
		&cobra.Command{
			Use:   "export inFile [outFileJSON]",
			Short: "Export bookmarks",
			Args:  cobra.RangeArgs(1, 2),
			Run:   wrapHandler(processExportBookmarksCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [outFile]",
			Short: "Remove bookmarks",
			Args:  cobra.RangeArgs(1, 2),
			Run:   wrapHandler(processRemoveBookmarksCommand),
		},
	)

	return cmd
}

func bookletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "booklet [description] outFile n inFile|imageFiles...",
		Short: "Arrange pages onto larger sheets of paper to make a booklet or zine",
		Long:  usageLongBooklet,
		Args:  cobra.MinimumNArgs(3),
		Run:   wrapHandler(processBookletCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func boxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boxes",
		Short: "List, add, remove page boundaries for selected pages",
		Long:  usageLongBoxes,
	}

	list := &cobra.Command{
		Use:   "list [boxTypes] inFile",
		Short: "List boxes",
		Args:  cobra.MinimumNArgs(1),
		Run:   wrapHandler(processListBoxesCommand),
	}
	list.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	list.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")

	add := &cobra.Command{
		Use:   "add description inFile [outFile]",
		Short: "Add boxes",
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processAddBoxesCommand),
	}
	add.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	add.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")

	remove := &cobra.Command{
		Use:   "remove boxTypes inFile [outFile]",
		Short: "Remove boxes",
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processRemoveBoxesCommand),
	}
	remove.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	cmd.AddCommand(list, add, remove)
	return cmd
}

func certificatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certificates",
		Short: "List, inspect, import, reset certificates",
		Long:  usageLongCertificates,
	}

	listOpts := &certificatesListOptions{json: false}
	list := &cobra.Command{
		Use:   "list",
		Short: "List certificates",
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processListCertificatesCommand(conf, args, listOpts)
		},
	}
	list.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	cmd.AddCommand(
		list,
		&cobra.Command{
			Use:   "inspect inFile",
			Short: "Inspect certificates",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processInspectCertificatesCommand),
		},
		&cobra.Command{
			Use:   "import inFile...",
			Short: "Import certificates",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processImportCertificatesCommand),
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Reset certificates",
			Run:   wrapHandler(resetCertificates),
		},
	)

	return cmd
}

func changeopwCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changeopw inFile opwOld opwNew",
		Short: "Change owner password",
		Long:  usageLongChangeOwnerPW,
		Args:  cobra.ExactArgs(3),
		Run:   wrapHandler(processChangeOwnerPasswordCommand),
	}
	return cmd
}

func changeupwCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changeupw inFile upwOld upwNew",
		Short: "Change user password",
		Long:  usageLongChangeUserPW,
		Args:  cobra.ExactArgs(3),
		Run:   wrapHandler(processChangeUserPasswordCommand),
	}
	return cmd
}

func collectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect inFile [outFile]",
		Short: "Create custom sequence of selected pages",
		Long:  usageLongCollect,
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processCollectCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process (required)")
	cmd.MarkFlagRequired("pages")
	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "List, reset configuration",
		Long:  usageLongConfig,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List configuration",
			Run:   wrapHandler(printConfiguration),
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Reset configuration",
			Run:   wrapHandler(resetConfiguration),
		},
	)

	return cmd
}

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create inFileJSON [inFile] outFile",
		Short: "Create PDF content including forms via JSON",
		Long:  usageLongCreate,
		Args:  cobra.RangeArgs(2, 3),
		Run:   wrapHandler(processCreateCommand),
	}
	return cmd
}

func cropCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crop description inFile [outFile]",
		Short: "Set cropbox for selected pages",
		Long:  usageLongCrop,
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processCropCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func cutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cut description inFile outDir [outFileName]",
		Short: "Custom cut pages horizontally or vertically",
		Long:  usageLongCut,
		Args:  cobra.MinimumNArgs(3),
		Run:   wrapHandler(processCutCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func decryptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt inFile [outFile]",
		Short: "Remove password protection",
		Long:  usageLongDecrypt,
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processDecryptCommand),
	}
	return cmd
}

func encryptCmd() *cobra.Command {
	opts := &encryptOptions{
		mode: "aes",
		key:  "256",
		perm: "none",
	}

	cmd := &cobra.Command{
		Use:   "encrypt inFile [outFile]",
		Short: "Set password protection",
		Long:  usageLongEncrypt,
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processEncryptCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "algorithm: rc4|aes")
	cmd.Flags().StringVarP(&opts.key, "key", "k", opts.key, "key length in bits: 40|128|256")
	cmd.Flags().StringVar(&opts.perm, "perm", opts.perm, "user access permissions: none|print|all")
	return cmd
}

func extractCmd() *cobra.Command {
	opts := &extractOptions{
		mode: "",
	}

	cmd := &cobra.Command{
		Use:   "extract inFile outDir",
		Short: "Extract images, fonts, content, pages or metadata",
		Long:  usageLongExtract,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processExtractCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "extraction mode: i(mage)|f(ont)|c(ontent)|p(age)|m(eta)")
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.MarkFlagRequired("mode")
	return cmd
}

func fontsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fonts",
		Short: "Install, list supported fonts, create cheat sheets",
		Long:  usageLongFonts,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List supported fonts",
			Run:   wrapHandler(processListFontsCommand),
		},
		&cobra.Command{
			Use:   "install fontFiles...",
			Short: "Install fonts",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processInstallFontsCommand),
		},
		&cobra.Command{
			Use:   "cheatsheet fontFiles...",
			Short: "Create font cheat sheets",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processCreateCheatSheetFontsCommand),
		},
	)

	return cmd
}

func formCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "form",
		Short: "List, remove fields, lock, unlock, reset, export, fill form via JSON or CSV",
		Long:  usageLongForm,
	}

	fill := &cobra.Command{
		Use:   "fill inFile inFileJSON [outFile]",
		Short: "Fill form with data",
		Args:  cobra.RangeArgs(2, 3),
		Run:   wrapHandler(processFillFormCommand),
	}
	fill.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	multifillOpts := &formMultifillOptions{mode: "single"}
	multifill := &cobra.Command{
		Use:   "multifill inFile inFileData outDir [outName]",
		Short: "Fill multiple form instances",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processMultiFillFormCommand(conf, args, multifillOpts)
		},
	}
	multifill.Flags().StringVarP(&multifillOpts.mode, "mode", "m", multifillOpts.mode, "output mode: single|merge")
	multifill.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile...",
			Short: "List form fields",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processListFormFieldsCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [outFile] <fieldID|fieldName>...",
			Short: "Remove form fields",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processRemoveFormFieldsCommand),
		},
		&cobra.Command{
			Use:   "lock inFile [outFile] [fieldID|fieldName]...",
			Short: "Lock form fields",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processLockFormCommand),
		},
		&cobra.Command{
			Use:   "unlock inFile [outFile] [fieldID|fieldName]...",
			Short: "Unlock form fields",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processUnlockFormCommand),
		},
		&cobra.Command{
			Use:   "reset inFile [outFile] [fieldID|fieldName]...",
			Short: "Reset form fields",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processResetFormCommand),
		},
		&cobra.Command{
			Use:   "export inFile [outFileJSON]",
			Short: "Export form data",
			Args:  cobra.RangeArgs(1, 2),
			Run:   wrapHandler(processExportFormCommand),
		},
		fill,
		multifill,
	)

	return cmd
}

func gridCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grid [description] outFile m n inFile|imageFiles...",
		Short: "Rearrange pages or images for enhanced browsing experience",
		Long:  usageLongGrid,
		Args:  cobra.MinimumNArgs(4),
		Run:   wrapHandler(processGridCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func imagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "List, extract, update images",
		Long:  usageLongImages,
	}

	list := &cobra.Command{
		Use:   "list inFile...",
		Short: "List images",
		Args:  cobra.MinimumNArgs(1),
		Run:   wrapHandler(processListImagesCommand),
	}
	list.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	extract := &cobra.Command{
		Use:   "extract inFile outDir",
		Short: "Extract images",
		Args:  cobra.ExactArgs(2),
		Run:   wrapHandler(processExtractImagesCommand),
	}
	extract.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	update := &cobra.Command{
		Use:   "update inFile imageFile [outFile] [objNr | (pageNr Id)]",
		Short: "Update images",
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processUpdateImagesCommand),
	}

	cmd.AddCommand(list, extract, update)
	return cmd
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [description] outFile imageFile...",
		Short: "Import/convert images to PDF",
		Long:  usageLongImportImages,
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processImportImagesCommand),
	}
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	cmd.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")
	return cmd
}

func infoCmd() *cobra.Command {
	opts := &infoOptions{fonts: false, json: false}
	cmd := &cobra.Command{
		Use:   "info inFile...",
		Short: "Print file info",
		Long:  usageLongInfo,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processInfoCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().BoolVar(&opts.fonts, "fonts", opts.fonts, "include font info")
	cmd.Flags().BoolVarP(&opts.json, "json", "j", opts.json, "output JSON")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func keywordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "List, add, remove keywords",
		Long:  usageLongKeywords,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List keywords",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListKeywordsCommand),
		},
		&cobra.Command{
			Use:   "add inFile keyword...",
			Short: "Add keywords",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processAddKeywordsCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [keyword...]",
			Short: "Remove keywords",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processRemoveKeywordsCommand),
		},
	)

	return cmd
}

func mergeCmd() *cobra.Command {
	opts := &mergeOptions{
		mode:        "create",
		sorted:      false,
		bookmarks:   false,
		dividerPage: false,
		optimize:    false,
	}

	cmd := &cobra.Command{
		Use:   "merge outFile inFile...",
		Short: "Concatenate PDFs",
		Long:  usageLongMerge,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			// Check if flags were explicitly set
			opts.bookmarksSet = cmd.Flags().Changed("bookmarks")
			opts.optimizeSet = cmd.Flags().Changed("optimize")
			processMergeCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "merge mode: create|append|zip")
	cmd.Flags().BoolVarP(&opts.sorted, "sort", "s", opts.sorted, "sort inFiles by file name")
	cmd.Flags().BoolVarP(&opts.bookmarks, "bookmarks", "b", opts.bookmarks, "create bookmarks")
	cmd.Flags().BoolVarP(&opts.dividerPage, "divider", "d", opts.dividerPage, "insert blank page between merged documents")
	cmd.Flags().BoolVar(&opts.optimize, "optimize", opts.optimize, "optimize before writing")
	return cmd
}

func ndownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ndown [description] n inFile outDir [outFileName]",
		Short: "Cut selected page into n pages symmetrically",
		Long:  usageLongNDown,
		Args:  cobra.MinimumNArgs(3),
		Run:   wrapHandler(processNDownCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func nupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nup [description] outFile n inFile|imageFiles...",
		Short: "Rearrange pages or images for reduced number of pages",
		Long:  usageLongNUp,
		Args:  cobra.MinimumNArgs(3),
		Run:   wrapHandler(processNUpCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func optimizeCmd() *cobra.Command {
	opts := &optimizeCommandOptions{fileStats: ""}
	cmd := &cobra.Command{
		Use:   "optimize inFile [outFile]",
		Short: "Optimize PDF by getting rid of redundant page resources",
		Long:  usageLongOptimize,
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processOptimizeCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.fileStats, "stats", opts.fileStats, "appends a stats line to a csv file")
	return cmd
}

func pagelayoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pagelayout",
		Short: "List, set, reset page layout for opened document",
		Long:  usageLongPageLayout,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List page layout",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListPageLayoutCommand),
		},
		&cobra.Command{
			Use:   "set inFile value",
			Short: "Set page layout",
			Args:  cobra.ExactArgs(2),
			Run:   wrapHandler(processSetPageLayoutCommand),
		},
		&cobra.Command{
			Use:   "reset inFile",
			Short: "Reset page layout",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processResetPageLayoutCommand),
		},
	)

	return cmd
}

func pagemodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pagemode",
		Short: "List, set, reset page mode for opened document",
		Long:  usageLongPageMode,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List page mode",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListPageModeCommand),
		},
		&cobra.Command{
			Use:   "set inFile value",
			Short: "Set page mode",
			Args:  cobra.ExactArgs(2),
			Run:   wrapHandler(processSetPageModeCommand),
		},
		&cobra.Command{
			Use:   "reset inFile",
			Short: "Reset page mode",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processResetPageModeCommand),
		},
	)

	return cmd
}

func pagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pages",
		Short: "Insert, remove selected pages",
		Long:  usageLongPages,
	}

	insertOpts := &pagesInsertOptions{mode: "before"}
	insert := &cobra.Command{
		Use:   "insert [description] inFile [outFile]",
		Short: "Insert pages",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processInsertPagesCommand(conf, args, insertOpts)
		},
	}
	insert.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	insert.Flags().StringVarP(&insertOpts.mode, "mode", "m", insertOpts.mode, "insertion mode: before|after")
	insert.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")

	remove := &cobra.Command{
		Use:   "remove inFile [outFile]",
		Short: "Remove pages",
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processRemovePagesCommand),
	}
	remove.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process (required)")
	remove.MarkFlagRequired("pages")

	cmd.AddCommand(insert, remove)
	return cmd
}

func paperCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "paper",
		Short: "Print list of supported paper sizes",
		Long:  usageLongPaper,
		Run:   wrapHandler(printPaperSizes),
	}
	return cmd
}

func permissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "List, set user access permissions",
		Long:  usageLongPerm,
	}

	set := &cobra.Command{
		Use:   "set inFile",
		Short: "Set permissions",
		Args:  cobra.ExactArgs(1),
		Run:   wrapHandler(processSetPermissionsCommand),
	}
	set.Flags().StringVar(&perm, "perm", "none", "user access permissions")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile...",
			Short: "List permissions",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processListPermissionsCommand),
		},
		set,
	)

	return cmd
}

func portfolioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "List, add, remove, extract portfolio entries",
		Long:  usageLongPortfolio,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List portfolio entries",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "add inFile file[,desc]...",
			Short: "Add portfolio entries",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processAddAttachmentsPortfolioCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [file...]",
			Short: "Remove portfolio entries",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processRemoveAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "extract inFile outDir [file...]",
			Short: "Extract portfolio entries",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processExtractAttachmentsCommand),
		},
	)

	return cmd
}

func posterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poster description inFile outDir [outFileName]",
		Short: "Create poster using paper size",
		Long:  usageLongPoster,
		Args:  cobra.MinimumNArgs(3),
		Run:   wrapHandler(processPosterCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func propertiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "properties",
		Short: "List, add, remove document properties",
		Long:  usageLongProperties,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List properties",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processListPropertiesCommand),
		},
		&cobra.Command{
			Use:   "add inFile nameValuePair...",
			Short: "Add properties",
			Args:  cobra.MinimumNArgs(2),
			Run:   wrapHandler(processAddPropertiesCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [name...]",
			Short: "Remove properties",
			Args:  cobra.MinimumNArgs(1),
			Run:   wrapHandler(processRemovePropertiesCommand),
		},
	)

	return cmd
}

func resizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resize description inFile [outFile]",
		Short: "Resize selected pages",
		Long:  usageLongResize,
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processResizeCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func rotateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate inFile rotation [outFile]",
		Short: "Rotate selected pages",
		Long:  usageLongRotate,
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processRotateCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	return cmd
}

func selectedpagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "selectedpages",
		Short: "Print definition of the -pages flag",
		Long:  usageLongSelectedPages,
		Run:   wrapHandler(printSelectedPages),
	}
	return cmd
}

func signaturesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signatures",
		Short: "Validate signatures",
		Long:  usageLongSignatures,
	}

	validateOpts := &signaturesValidateOptions{all: false, full: false}
	validate := &cobra.Command{
		Use:   "validate inFile",
		Short: "Validate signatures",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processValidateSignaturesCommand(conf, args, validateOpts)
		},
	}
	validate.Flags().BoolVarP(&validateOpts.all, "all", "a", validateOpts.all, "validate all signatures")
	validate.Flags().BoolVarP(&validateOpts.full, "full", "f", validateOpts.full, "comprehensive output")
	validate.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	cmd.AddCommand(validate)
	return cmd
}

func splitCmd() *cobra.Command {
	opts := &splitOptions{
		mode: "span",
	}

	cmd := &cobra.Command{
		Use:   "split inFile outDir [span|pageNr...]",
		Short: "Split up a PDF by span or bookmark",
		Long:  usageLongSplit,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processSplitCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "split mode: span|bookmark|page")
	return cmd
}

func stampCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stamp",
		Short: "Add, remove, update stamps",
		Long:  usageLongStamp,
	}

	// Add subcommand
	addOpts := &stampOptions{mode: "text"}
	add := &cobra.Command{
		Use:   "add string|file description inFile [outFile]",
		Short: "Add stamps",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processAddStampsCommand(conf, args, addOpts)
		},
	}
	add.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	add.Flags().StringVarP(&addOpts.mode, "mode", "m", addOpts.mode, "stamp mode: text|image|pdf")
	add.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	add.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	// Update subcommand
	updateOpts := &stampOptions{mode: "text"}
	update := &cobra.Command{
		Use:   "update string|file description inFile [outFile]",
		Short: "Update stamps",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processUpdateStampsCommand(conf, args, updateOpts)
		},
	}
	update.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	update.Flags().StringVarP(&updateOpts.mode, "mode", "m", updateOpts.mode, "stamp mode: text|image|pdf")
	update.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	update.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	remove := &cobra.Command{
		Use:   "remove inFile [outFile]",
		Short: "Remove stamps",
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processRemoveStampsCommand),
	}
	remove.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	cmd.AddCommand(add, update, remove)
	return cmd
}

func trimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trim inFile [outFile]",
		Short: "Create trimmed version of selected pages",
		Long:  usageLongTrim,
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processTrimCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process (required)")
	cmd.MarkFlagRequired("pages")
	return cmd
}

func validateCmd() *cobra.Command {
	opts := &validateOptions{
		mode:     "relaxed",
		links:    false,
		optimize: false,
	}

	cmd := &cobra.Command{
		Use:   "validate inFile...",
		Short: "Validate PDF against PDF 32000-1:2008 (PDF 1.7)",
		Long:  usageLongValidate,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processValidateCommand(conf, args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "validation mode: strict|relaxed")
	cmd.Flags().BoolVarP(&opts.links, "links", "l", opts.links, "check for broken links")
	cmd.Flags().BoolVar(&opts.optimize, "optimize", opts.optimize, "optimize resources")
	cmd.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")
	return cmd
}

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Long:  usageLongVersion,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "pdfcpu: %s\n", version)
			fmt.Fprintf(os.Stdout, "commit: %s\n", commit)
			fmt.Fprintf(os.Stdout, "date: %s\n", date)
		},
	}
	return cmd
}

func viewerprefCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewerpref",
		Short: "List, set, reset viewer preferences",
		Long:  usageLongViewerPreferences,
	}

	listOpts := &viewerpreferencesListOptions{all: false, json: false}
	list := &cobra.Command{
		Use:   "list inFile",
		Short: "List viewer preferences",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processListViewerPreferencesCommand(conf, args, listOpts)
		},
	}
	list.Flags().BoolVarP(&listOpts.all, "all", "a", listOpts.all, "output all (including default values)")
	list.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	cmd.AddCommand(
		list,
		&cobra.Command{
			Use:   "set inFile (inFileJSON | JSONstring)",
			Short: "Set viewer preferences",
			Args:  cobra.ExactArgs(2),
			Run:   wrapHandler(processSetViewerPreferencesCommand),
		},
		&cobra.Command{
			Use:   "reset inFile",
			Short: "Reset viewer preferences",
			Args:  cobra.ExactArgs(1),
			Run:   wrapHandler(processResetViewerPreferencesCommand),
		},
	)

	return cmd
}

func watermarkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watermark",
		Short: "Add, remove, update watermarks",
		Long:  usageLongWatermark,
	}

	// Add subcommand
	addOpts := &watermarkOptions{mode: "text"}
	add := &cobra.Command{
		Use:   "add string|file description inFile [outFile]",
		Short: "Add watermarks",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processAddWatermarksCommand(conf, args, addOpts)
		},
	}
	add.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	add.Flags().StringVarP(&addOpts.mode, "mode", "m", addOpts.mode, "watermark mode: text|image|pdf")
	add.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	add.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	// Update subcommand
	updateOpts := &watermarkOptions{mode: "text"}
	update := &cobra.Command{
		Use:   "update string|file description inFile [outFile]",
		Short: "Update watermarks",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			conf := getConfig()
			if conf.Version != model.VersionStr {
				model.CheckConfigVersion(conf.Version)
			}
			processUpdateWatermarksCommand(conf, args, updateOpts)
		},
	}
	update.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	update.Flags().StringVarP(&updateOpts.mode, "mode", "m", updateOpts.mode, "watermark mode: text|image|pdf")
	update.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	update.Flags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")

	remove := &cobra.Command{
		Use:   "remove inFile [outFile]",
		Short: "Remove watermarks",
		Args:  cobra.RangeArgs(1, 2),
		Run:   wrapHandler(processRemoveWatermarksCommand),
	}
	remove.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")

	cmd.AddCommand(add, update, remove)
	return cmd
}

func zoomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zoom description inFile [outFile]",
		Short: "Zoom in/out of selected pages",
		Long:  usageLongZoom,
		Args:  cobra.MinimumNArgs(2),
		Run:   wrapHandler(processZoomCommand),
	}
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints)|in(ches)|cm|mm")
	return cmd
}

func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for pdfcpu.

To load completions:

Bash:

  $ source <(pdfcpu completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ pdfcpu completion bash > /etc/bash_completion.d/pdfcpu
  # macOS:
  $ pdfcpu completion bash > $(brew --prefix)/etc/bash_completion.d/pdfcpu

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ pdfcpu completion zsh > "${fpath[1]}/_pdfcpu"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ pdfcpu completion fish | source

  # To load completions for each session, execute once:
  $ pdfcpu completion fish > ~/.config/fish/completions/pdfcpu.fish

PowerShell:

  PS> pdfcpu completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> pdfcpu completion powershell > pdfcpu.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
	return cmd
}
