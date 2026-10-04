package patch

import "github.com/dkotik/cuebook"

// Match locates the entry that was just updated by the last patch.
func (r Result) Match(difference []byte) (entry cuebook.Entry, ok bool) {
	if r.LastChange == nil {
		return
	}
	return
}
