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
		Name:  "sdf"
		Email: "sdffsd@perfect.com"
	},
  {
		Name:  "First11111aa"
		Email: "test1@testdomain"
	}
]
