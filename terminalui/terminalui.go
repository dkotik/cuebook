/*
Package terminalui provides user interface built on Charm.
*/
package terminalui

import (
	"github.com/dkotik/cuebook/terminalui/internal"
	"github.com/dkotik/cuebook/terminalui/internal/file"
)

// FilePickerConfiguration configures [NewFilePicker].
type FilePickerConfiguration = file.Configuration

// LoadFileEvent asks the file picker to read the file at the given path.
type LoadFileEvent = file.LoadEvent

var NewFilePicker = file.New

var WithStateEventTransformers = internal.WithStateEventTransformers

var ParseFileToBookAndCreateEntryListIfNeeded = internal.ParseFileToBookAndCreateEntryListIfNeeded
