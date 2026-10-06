+++
title = 'Data model and annotations'
description = 'Reference for Book, Entry, Field, and the Cuebook CUE attributes.'
weight = 30
+++

The root package models a CUE list as a `Book`. Entries are CUE structs, and their concrete fields are exposed as `Field` values.

## Book

Create a book with `cuebook.New(source)`, where `source` is a byte slice containing CUE. The returned `Book` embeds the underlying `cue.Value` and provides convenience methods for its list:

- `Len()` returns the number of list values.
- `GetValue(index)` returns the CUE value at a list index.
- `GetField(entryIndex, fieldIndex)` gets a field from an entry. Regular fields come before detail fields.
- `EachValue()` iterates over the list's `cue.Value` items.
- `EachEntry()` iterates as `(Entry, error)` pairs, converting each item to a struct entry.

The package also provides the iterator helpers `cuebook.EachValue(value)`, `cuebook.EachField(value, options...)`, and `cuebook.EachFieldDefinition(value)`. They are useful when working directly with CUE values, including schema field definitions.

## Entry

An `Entry` retains its underlying CUE value in `Value` and exposes its concrete fields in two slices:

- `Fields` contains ordinary fields.
- `Details` contains fields marked with `@cuebook(detail)`.

`GetTitle()` returns the entry title. A field marked `@cuebook(title)` supplies it; if no title is marked, Cuebook falls back to the string representation of the first ordinary field. `GetDescription()` returns ordinary field values as strings and omits the first value when it duplicates the title.

Use `GetField(index)` to access a field by position across both slices, or `GetFieldByName(name)` to look it up by name. `EachEntry` can return an error for a list item that is not a concrete struct, so check the error yielded for every item.

## Field

A `Field` has a `Name` and an underlying CUE `Value`. `String()` provides its display representation, `Default()` reports an available default value, and `WithValue(text)` creates a CUE AST field with the supplied text formatted according to supported Cuebook attributes. These helpers do not modify the original `Book` or source bytes.

## `@cuebook` attributes

Cuebook reads attributes attached to CUE fields:

| Attribute | Effect |
| --- | --- |
| `@cuebook(title)` | Marks the field as the entry title. The field remains in `Fields` when concrete. |
| `@cuebook(detail)` | Places the concrete field in `Entry.Details` instead of `Entry.Fields`. |
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
