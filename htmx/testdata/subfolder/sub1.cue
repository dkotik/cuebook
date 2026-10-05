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
		Name:  "First11111aaDDD" @cuebook(title)
		Email: "test1@testdomain.com"
	},
  {
		Name:  "2378"
		Email: "sdf@sdf.com"
	}
]
