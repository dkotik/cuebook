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
		Name:  "First11111aa"
		Email: "test1@testdomain"
	},
  {
		Name:  "First11111aaDDD"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "First11111aa1"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "1 really great"
		Email: "1dfjd@kekeke.kekeke"
	},
  {
		Name:  "2378"
		Email: "sdf@sdf.com"
	}
]
