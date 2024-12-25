// # Example File
//
// First comment is a description of the contents in Markdown format.
// The first heading is the title of the document.
//
// `cue export --out=yaml simple.cue`

#email: =~ "^[^@]+@[^@]+$"

[...{
	Name:   string
	Email:  #email | [...#email]
	Notes?: string
}] & [
  {
  	Name: "First"
  	Email: "test@testdomain.com"
  },
  {
  	Name: "Second"
  	Email: [
      "test@testdomain.com",
      "another@sdfklsdjf.com"
    ]
    Notes: "sdfksjdflk\nsfsdfsdfasjdflksjdf\nsdfsdf"
  },
]
