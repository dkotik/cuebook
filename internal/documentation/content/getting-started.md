+++
title = 'Getting started'
description = 'Parse and inspect a CUE list with the Cuebook Go package.'
weight = 20
+++

## Install

From a Go module, add Cuebook as a dependency:

```sh
go get github.com/dkotik/cuebook
```

The examples below use Go's range-over-function syntax for iterators, available in Go 1.23 and later.

## Write a CUE document

A Cuebook document contains a top-level list. CUE definitions can describe and validate the list's entries alongside the data:

```cue
#Contact: {
    Name:  string @cuebook(title)
    Email: =~"^[^@]+@[^@]+$"
    Notes?: string @cuebook(detail)
    ...
}

[...#Contact] & [
    {Name: "Ada Lovelace", Email: "ada@example.test"},
    {Name: "Grace Hopper", Email: "grace@example.test", Notes: "Prefers email"},
]
```

The `#Contact` definition constrains each item. The `...` keeps the struct open to additional fields; remove it if entries should only contain the declared fields. The list unifies the definition with the concrete data, so CUE validates each contact.

## Parse and iterate in Go

`cuebook.New` accepts the source as bytes and returns a validated `Book`. It does not open a path or write back to the source.

```go
package main

import (
    "fmt"
    "log"

    "github.com/dkotik/cuebook"
)

func main() {
    source := []byte(`
#Contact: {
    Name:  string @cuebook(title)
    Email: =~"^[^@]+@[^@]+$"
    Notes?: string @cuebook(detail)
    ...
}

[...#Contact] & [
    {Name: "Ada Lovelace", Email: "ada@example.test"},
    {Name: "Grace Hopper", Email: "grace@example.test", Notes: "Prefers email"},
]
`)

    book, err := cuebook.New(source)
    if err != nil {
        log.Fatal(err)
    }

    count, err := book.Len()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Contacts: %d\n", count)

    for entry, err := range book.EachEntry() {
        if err != nil {
            log.Fatal(err)
        }
        fmt.Println(entry.GetTitle())
        for _, field := range entry.Fields {
            fmt.Printf("  %s: %s\n", field.Name, field.String())
        }
    }
}
```

An error from `New` means the CUE source could not be compiled or validated as a list. `EachEntry` reports conversion errors separately for individual list items, allowing callers to handle an invalid entry while iterating.

## Next steps

- Learn how titles, regular fields, and detail fields are represented in [the data model]({{< relref "data-model.md" >}}).
- See how to obtain source locations with [byte ranges]({{< relref "source-ranges.md" >}}).
