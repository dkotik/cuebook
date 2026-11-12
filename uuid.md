# Add Identifier Latching to Patch Operations

The following methods produce a patch that updates a Cue file. The patch looks match a byte sequence using a `ByteAnchor`:

- func (e Entry) AddField(name string, kind FieldKind, value string) (Patch, error)
- func (e Entry) DeleteFromStructList() (Patch, error)
- func (e Entry) MergeFieldValues(values map[string]string) (Patch, error)
- func (e Entry) MergeFieldValuesAndRemove(values map[string]string, removals []string) (Patch, error)
- func (e Entry) RemoveFieldValue(fieldName string) (Patch, error)
- func (e Entry) ReplaceStructListEntry(replacement []byte) (Patch, error)
- func (e Entry) SwapFields(a, b string) (Patch, error)
- func (e Entry) UpdateFieldValue(fieldName, value string) (Patch, error)
- func (book Book) MoveEntry(from, to int) (Patch, error)
- func (book Book) SwapEntries(a, b Entry) (Patch, error)

Such operations work alright on flexible free-form data. However, if an `Entry` has an identifier, it would be far more precise to match the entry by looking for the same identifier value as `Entry.UUID` by the same name as the `Book.entryUniqueIdentifierFieldName` instead of matching by `ByteAnchor`.

# Implementation Plan

For each method in the list above, produce a version that emits a new kind of patch, which does not match by raw bytes but by value of a unique identifier.

1. Add a strict identity resolver first. Define an entry identity as the configured field name plus its non-empty string value, and resolve it against a freshly parsed source when applying a patch. Require exactly one match: missing IDs should conflict, duplicate IDs should be ambiguous, and an identified patch should never fall back to byte matching. The resolver can still return the entry’s current byte range for splicing; bytes locate the edit only after identity has selected the entry.

2. Make identifier patches immutable desired-state transitions. Store the entry identity, operation details, and enough before/after values to support conflict detection and Invert(). On apply, make the change if the expected before-state is present, return unchanged if the after-state is already present, and conflict otherwise. This also exposes an API question: Patch.Difference() and Result.Match currently rely on ByteAnchor; consider how history and UI selection should represent an identity-based change. Decide explicitly whether operations may change the UUID field—updating, removing, swapping, or replacing it can otherwise make the target unresolvable.

3. Migrate in stages, with dedicated identity tests. Start with single-entry field operations, then deletion/replacement and field swaps, and finish with MoveEntry and SwapEntries. For moves, resolve the source and destination indices to entry identities when constructing the patch; don’t reuse indices against a changed source. Test retries, inverses, unrelated edits around a target, changed or missing entries, duplicate IDs, and fallback to existing byte-anchor behavior when an entry has no identifier.

4. Carry the identifier field name on each Entry. The listed patch-producing methods have an Entry receiver, but the configured field name currently belongs to Book. Retain that name alongside Entry.UUID when Book.EachEntry() constructs entries, so those methods can create identity-based patches without trying to rediscover the schema or relying on a global. This also gives patches the complete identity pair: field name and value.

5. Treat the identifier field as immutable in ordinary patch operations. Reject updates, removals, or swaps that would change it, and require ReplaceStructListEntry to preserve its current value. If changing an ID is a supported operation, give it a dedicated re-key patch that stores both the old and new identities; otherwise retries and inverses can lose track of the entry.
