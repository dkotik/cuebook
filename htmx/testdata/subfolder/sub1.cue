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
		Name:  "2378"
		Email: "sdf@sdf.com"
	},
  {
		Name:  "third and final"
		Email: "third@some.email"
	},
  {
		Name:  "First11111aaDDD1"
		Email: "test1@testdomain.com"
	}
]
