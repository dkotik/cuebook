// Package patch preserves the former patch-package API while forwarding all
// behavior to the root cuebook package.
//
// Deprecated: use github.com/dkotik/cuebook directly.
package patch

import (
	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
)

type Patch = cuebook.Patch
type Result = cuebook.Result
type Error = cuebook.PatchError
type Localizable = cuebook.Localizable

const (
	ErrUnknown                   = cuebook.ErrUnknown
	ErrSourceIsNotList           = cuebook.ErrSourceIsNotList
	ErrByteRangesOverlap         = cuebook.ErrByteRangesOverlap
	ErrUnableToFormatFieldValue  = cuebook.ErrUnableToFormatFieldValue
	ErrEntryNotStruct            = cuebook.ErrEntryNotStruct
	ErrTargetFieldNotStructField = cuebook.ErrTargetFieldNotStructField
	ErrDoubleCommaAtListEnd      = cuebook.ErrDoubleCommaAtListEnd
	ErrNotListOfStructs          = cuebook.ErrNotListOfStructs
	ErrRootListEndNotFound       = cuebook.ErrRootListEndNotFound
)

// ErrByteRangeNotFound is the core cuebook byte-range lookup sentinel.
var ErrByteRangeNotFound = cuebook.ErrByteRangeNotFound

func Commit(targetPath, swapPath string, p Patch) (Result, error) {
	return cuebook.Commit(targetPath, swapPath, p)
}

func Validated(p Patch) Patch {
	return cuebook.Validated(p)
}

func AppendToStructList(source []byte, value cue.Value) (Patch, error) {
	return cuebook.AppendToStructList(source, value)
}

func DeleteFromStructList(source []byte, value cue.Value) (Patch, error) {
	return cuebook.DeleteFromStructList(source, value)
}

func SwapEntries(source []byte, a, b cue.Value) (Patch, error) {
	return cuebook.SwapEntries(source, a, b)
}

func UpdateRange(source []byte, r cuebook.ByteRange, replacement []byte) (Patch, error) {
	return cuebook.UpdateRange(source, r, replacement)
}

func ReplaceStructListEntry(source []byte, value cue.Value, replacement []byte) (Patch, error) {
	return cuebook.ReplaceStructListEntry(source, value, replacement)
}

func UpdateFieldValue(source []byte, entry, field cue.Value, value string) (Patch, error) {
	return cuebook.UpdateFieldValue(source, entry, field, value)
}

func RemoveFieldValue(source []byte, entry cue.Value, fieldName string) (Patch, error) {
	return cuebook.RemoveFieldValue(source, entry, fieldName)
}

func MergeFieldValues(source []byte, entry cue.Value, values map[string]string) (Patch, error) {
	return cuebook.MergeFieldValues(source, entry, values)
}
