+++
title = 'Source byte ranges'
description = 'Locate CUE values in source and re-identify their content after edits.'
weight = 40
+++

Cuebook keeps each entry connected to the CUE syntax node from which it was compiled. The package's byte-range helpers expose that source location for applications that need to target an entry when making edits.

## Locate a value

`cuebook.NewByteRange(value)` returns a `ByteRange` for the first concrete expression with source information. `Head` is the starting byte offset and `Tail` is the ending byte offset; the range is half-open, so it includes `source[Head:Tail]`. Offsets count bytes, not Unicode characters.

```go
entryRange, err := cuebook.NewByteRange(entry.Value)
if err != nil {
    // The CUE value has no concrete source expression to locate.
    return err
}

originalText := source[entryRange.Head:entryRange.Tail]
_ = originalText
```

A range is tied to the exact source version that produced it. Do not apply its offsets to changed bytes without locating the content again. Values that were constructed without concrete source information may not have a range; in that case `NewByteRange` returns `cuebook.ErrByteRangeNotFound`.

## Anchor content across changes

`ByteRange.Anchor(source)` captures the bytes in the range and counts identical occurrences before it. A `ByteAnchor` can then search a newer source with `Match(source)` and skip that many preceding duplicates. This is a content-based anchor, not a globally unique identifier: if the source changes or matching duplicates are inserted or removed, the caller should handle `ErrByteRangeNotFound` and verify the matched result.

`ByteRange.PreceedingEntryAnchor(source)` captures nearby preceding content to help identify an entry relative to its context. Both helpers are useful when source can be edited between reading an entry and applying a change.

The patching workflow lives in the separate `github.com/dkotik/cuebook/patch` package. The root package only reports locations and anchors; it does not alter or persist source files.

[Back to getting started]({{< relref "getting-started.md" >}}).
