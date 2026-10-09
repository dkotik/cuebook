// # Core1 Title
//
// Some markdown contents is here.
//
// ---
//
// File details...
//
#email: =~"^[^@]+@[^@]+$"
#contact: {
	// current definition scanner cannot detect abstract definitions yet

	// Name help text.
	Name: string @cuebook(title)
	// Email help text.
	Email: #email | [...#email]
	Notes?:    string @cuebook(detail, multiline)
	Password?: string @cuebook(detail,trim,argon2id)
	... // allow any additional fields
}

[...#contact] & [
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
		Notes: "sfs sdf sdfsdf"
	},
  {
		Name:  "new entry" @cuebook(title)
		Email: "test1@testdomain.com"
		Notes: "sdfsdf"
	},
  {
		Name:  "First11111aa1"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "dsf lsadjl fksjdfl ksjdflkj"
		Email: "sdkfjsdlkfjsdlkfj2@sdfsdaf.cdsom"
		Notes: "skdjfksjd fs"
	},
  {
		Name:  "new entry" @cuebook(title)
		Email: "test1@doodod.co"
	},
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	},
  {
		Name:  "23424n33"
		Email: "234234@dsf.com"
	},
  {
		Name:  "Adam Bink 2"
		Email: "sdfjkh@sdfs1df.com"
	},
  {
		Name:  "new entry"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "Maria" @cuebook(title)
		Email: "mr@sfsdf.com"
	},
  {
		Name:  "Maria" @cuebook(title)
		Email: "mr@sfsdf.com"
	},
  {
		Name:  "xcxcx" @cuebook(title)
		Email: "cvxvxcv@ds1afsd.com"
	},
  {
		Name:  "xcxcx" @cuebook(title)
		Email: "cvxvxcv@ds1afsd.com"
	}
]
