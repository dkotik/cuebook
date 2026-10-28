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
		Name:  "Maria" @cuebook(title)
		Email: "mr@sfsdf.com"
	},
  {
		Name:  "First11111aa1" @cuebook(title)
		Email: "test1@testdo1main.com"
		Notes: """
			sd fsdf 
			sdf
			s
			adf sdf
			sdf
			sdfs
			dfsdf

			"""
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
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	},
  {
		Name:  "xcxcx" @cuebook(title)
		Email: "cvxvxcv@ds1afsd.com"
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
		Name:     "slkdjf"
		Email:    "sdklfsjad@sdfsadf.com"
		Notes:    "sdf"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$FyJ1ijloeHPnOzbUd+BKyA$VVG0+6ajCcBQ74S/7sMLBbWz1/P8HuvYr/l99r7FQqE"
	},
  {
		Name:  "sdfjlsdf"
		Email: "lksjdfljsd@sdf"
	}
]
