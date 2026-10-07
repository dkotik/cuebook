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
		Name:  "Maria"
		Email: "mr@sfsdf.com"
	},
  {
		Name:  "aFirst11111aaDDD"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "2378" @cuebook(title)
		Email: "sdf@sdf.com1"
	},
  {
		Name:  "Super Rambo 123" @cuebook(title)
		Email: "test1@testdomain.com"
	},
  {
		Name:  "12sdf33zz"
		Email: "sdffsd@perfect.com"
	},
  {
		Name:  "1 really great21"
		Email: "1dfjd@kekeke.kekeke"
	}
]
