// # Core1 Title
//
// Some markdown contents is here.
#email: =~"^[^@]+@[^@]+$"
#contact: {
	// current definition scanner cannot detect abstract definitions yet
	Name: string @cuebook(title)
	Email: #email | [...#email]
	Notes?:    string @cuebook(detail)
	Password?: string @cuebook(detail,trim,argon2id)
	... // allow any additional fields
}

[...#contact] & [
  {
		Name:  "new entry"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "First11111aa1"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "First11111aa1axx"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "23424"
		Email: "234234@31234.com"
	},
  {
		Name:  "sdjfl ksjdflk"
		Email: "sdfjkh@sdfsdf.com"
	}
]
