---
title: 'Data model and annotations'
description: 'Reference for Book, Entry, Field, and the Cuebook CUE attributes.'
weight: 30
---

The root package models a CUE list as a `Book`. Entries are CUE structs, and their concrete fields are exposed as `Field` values.

## Book

Create a book with `cuebook.New(source)`, where `source` is a byte slice containing CUE. The returned `Book` embeds the underlying `cue.Value` and provides convenience methods for its list:

- `Len()` returns the number of list values.
- `GetValue(index)` returns the CUE value at a list index.
- `GetField(entryIndex, fieldIndex)` gets a field from an entry. Regular fields come before detail fields.
- `EachEntry()` iterates as `(Entry, error)` pairs, converting each item to a struct entry.

For direct iteration over the underlying CUE list, call `Book.Value.List()` and check its returned error. The package also provides `cuebook.EachFieldDefinition(value)` to inspect abstract schema fields.

## Entry

An `Entry` retains its underlying CUE value in `Value` and exposes its concrete fields in two slices:

- `Fields` contains ordinary fields.
- `Details` contains fields marked with `@cuebook(detail)`.

`Title` exposes the entry title. A field marked `@cuebook(title)` supplies it; if no title is marked, Cuebook falls back to the string representation of the first ordinary field. `GetDescription()` returns ordinary field values as strings and omits the first value when it duplicates the title.

Use `GetField(index)` to access a field by position across both slices, or `GetFieldByName(name)` to look it up by name. `EachEntry` can return an error for a list item that is not a concrete struct, so check the error yielded for every item.

## Field

A `Field` has a `Name` and an underlying CUE `Value`. `String()` provides its display representation, `Default()` reports an available default value, and `WithValue(text)` creates a CUE AST field with the supplied text formatted according to supported Cuebook attributes. These helpers do not modify the original `Book` or source bytes.

## `@cuebook` attributes

Cuebook reads attributes attached to CUE fields:

| Attribute | Effect |
| --- | --- |
| `@cuebook(title)` | Marks the field as the entry title. The field remains in `Fields` when concrete. |
| `@cuebook(detail)` | Places the concrete field in `Entry.Details` instead of `Entry.Fields`. |
| `@cuebook(uuid)` | Marks a schema field as the entry identifier used for UUID-based patch targeting. |
| `@cuebook(multiline)` | Marks a field as multiline for applications that present or edit it. |
| `@cuebook(default=UUID)` | Provides a UUID default through the field helpers. |
| `@cuebook(default=SFID)` | Provides a Snowflake-style ID default; query parameters can configure its encoding and node. |
| `@cuebook(trim)` | Trims surrounding whitespace when formatting a submitted value. |
| `@cuebook(argon2id)` | Applies Argon2id hashing when formatting a submitted value. |

For example, an optional identifier can be kept out of the ordinary field list while still receiving a generated default:

```cue
ID?: string @cuebook(default=UUID,detail)
```

The attribute is metadata; CUE constraints still define which values are valid. For example, use a CUE type or pattern constraint to validate an email address rather than relying on a display annotation.

[Continue to source byte ranges]({{< relref "source-ranges.md" >}}).

## Unique Entry Identifiers

Mark an entry schema field with `@cuebook(uuid)` to configure it as the entry identifier. The first field definition carrying this attribute is used. For example:

```cue
#entry: {
    ID?: string @cuebook(uuid)
    Name: string
}
[...#entry] & [
  {
    ID: "unique-id-1",
    Name: "Write tests"
  }
]
```

The `ID` field is populated only when an entry actually has a non-empty string value in the configured field. A missing, empty, or non-string value leaves `UUID` empty; parsing an entry does not generate an identifier. The annotation identifies the schema field but does not make it required or validate its value.

Identifier uniqueness is not enforced. Any changes to a list item that has a `UUID` field will similarly update all other items in the same list with the same `UUID`.

### Patch targeting

When changes are applied to entries with a `UUID` field, an identifier-based patch parses the current CUE source and resolves the configured field name and UUID.

Ordinary field operations preserve identifier immutability. Updating or removing the identifier, swapping it with another field, or replacing an entry with a different or missing identifier is rejected.

There is currently no re-key operation. Ordinary entry patches cannot change an identifier. If it must be changed, edit the source file.
