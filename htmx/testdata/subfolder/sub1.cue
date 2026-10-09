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
		Name:  "aFirst11111aaDDD" @cuebook(title)
		Email: "test1@testdo1main.com"
	},
  {
		Name:  "2378" @cuebook(title)
		Email: "sdf@sdf.com1"
	},
  {
		Name:  "12sdf33zz"
		Email: "sdffsd@perfect.com"
	},
  {
		Name:  "1 really great21"
		Email: "1dfjd@kekeke.kekeke"
	},
  {
		Name:  "Maria"
		Email: "mr@sfsdf.com"
	},
  {
		Name:  "First11111aa1axx44"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "new entry"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "Super Rambo 123" @cuebook(title)
		Email: "test1@testdom1ain.com"
	},
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	}
]
