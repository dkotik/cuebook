package cuebook

import "testing"

func TestMarkdownCommentParsing(t *testing.T) {
	source := []byte(`
	// # comment 1
	//
	// comment 2

	// any later comments are disregarded
	[
		{ "key": "value" },
		{ "key2": "value2" }
	]`)

	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	for _, comment := range book.Doc() {
		t.Log("comments:", comment.Text())
		// t.Log("comments:", comment.Comments())
		t.Log("byte range:", comment.End().Offset())
		t.Log("comment chopped:", string(source[:comment.End().Offset()]))
	}

	meta := book.Metadata()
	if meta.Node == nil {
		t.Fatal("empty first comment")
	}
	if meta.Title() != "comment 1" {
		t.Fatal("unexpected title")
	}
	if meta.Description() != "comment 2\n" {
		t.Fatal("unexpected description")
	}

	list, err := book.List()
	if err != nil {
		t.Fatal(err)
	}

	// for _, comment := range list.Doc() {
	// 	t.Log("comments:", comment.Text())
	// 	t.Log("comments:", comment.Comments())
	// 	t.Log("byte range:", comment.End().Offset())
	// 	t.Log("comment chopped:", string(source[:comment.End().Offset()]))
	// }

	for list.Next() {
		item := list.Value()
		for _, comment := range item.Doc() {
			t.Log("comments:", comment.Text())
			t.Log("comments:", comment.Comments())
			t.Log("byte range:", comment.End().Offset())
			t.Log("comment chopped:", string(source[:comment.End().Offset()]))
		}
	}

	// t.Fatal("impl")
}
