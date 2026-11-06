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
	Notes?:    string @cuebook(detail,multiline)
	Password?: string @cuebook(trim,argon2id,locked)
	... // allow any additional fields
}

[...#contact] & [
  {
		Name:  "Maria12"
		Email: "mr121@sfsdf.com"
		Notes: """
			# laskdjasldkj asd

			sdlfkjsdflkj



			sdf sd fsdf

			dsfsdf



			> sdfs dlfkjsdflksajdfsljdfk



			- 1
			- **2**

			# testing test

			sdfsdf



			- \\[ \\] sdfsadfsdf
			- \\[ \\] sdfsdf


			"""
	},
  {
		Name:  "dsf lsadjl fksjdfl ksjdflkj"
		Email: "sdkfjsdlkfjsdlkfj2@sdfsdaf.cdsom"
		Notes: "skdjfksjd fs"
	},
  {
		Name:  "new entry"
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
		Name:  "23424n33"
		Email: "234234@31234a1.com"
	},
  {
		Name:  "xcxcx"
		Email: "cvxvxcv@ds1afsd.com"
	},
  {
		Name:  "xcxcx"
		Email: "cvxvxcv@ds1afsd.com"
	},
  {
		Name:  "Maria"
		Email: "mr@sfsdf.com"
	},
  {
		Name:     "12slkdjf"
		Email:    "sdklfsjad@sdfsadf.com"
		Notes:    "sdf"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$FyJ1ijloeHPnOzbUd+BKyA$VVG0+6ajCcBQ74S/7sMLBbWz1/P8HuvYr/l99r7FQqE"
	},
  {
		Name:  "sdfjlsdf"
		Email: "lksjdfljsd@sdf"
	},
  {
		Name:       "1 really great21"
		Email:      "1dfjd@kekeke.kekeke1"
		Notes:      "dfgdf"
		Introduced: true
	},
  {
		Name:  "Someone!!!!"
		Email: "someEmail@somehost.net"
	},
  {
		Name:  "nnnnnnew"
		Email: "nnnnnnew@new.com"
		Notes: "sdfjksdf"
	}
]
