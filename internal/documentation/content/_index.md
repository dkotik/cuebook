+++
title = 'Cuebook Go package'
description = 'A Go library for parsing and working with structured CUE lists.'
+++

Cuebook is a Go package for reading structured entries from [CUE](https://cuelang.org/) documents. It compiles and validates a CUE source, then exposes its list entries and fields as Go values while retaining their connection to the original source.

CUE lets a document carry both its data and the constraints that describe valid data. A Cuebook document is typically a top-level list of structs, with a CUE definition providing the shared schema:

```cue
#Task: {
    Title: string @cuebook(title)
    Done:  bool
    Notes?: string @cuebook(detail)
    ...
}

[...#Task] & [
    {Title: "Prepare release", Done: false, Notes: "Confirm the checklist"},
    {Title: "Publish notes", Done: true},
]
```

The root `cuebook` package parses and validates bytes; it does not read files or save edits. Applications can use its `Book`, `Entry`, and `Field` types to inspect the document. Other Cuebook packages provide workflows such as patching and user interfaces.

## Guides

- [Getting started]({{< relref "getting-started.md" >}}) — install the module, parse a CUE list, and iterate over entries.
- [Data model and annotations]({{< relref "data-model.md" >}}) — understand books, entries, fields, and `@cuebook` attributes.
- [Source byte ranges]({{< relref "source-ranges.md" >}}) — locate values in source and anchor them across edits.

## What `cuebook.New` checks

`cuebook.New` compiles the supplied bytes with CUE, validates the resulting value, and requires a top-level list. It returns an error if compilation or validation fails or if the top-level value is not a list. `EachEntry` then converts individual list values to structured entries; each item must be a concrete CUE struct for that conversion to succeed.
