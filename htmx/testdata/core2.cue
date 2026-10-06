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
		Name:  "First11111aa1"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "sdkjfhsdkjfh"
		Email: "12312312@2324.2343"
	},
  {
		Name:  "third and finaleee"
		Email: "third@some.email"
	},
  {
		Name:  "Adam Bink"
		Email: "sdfjkh@sdfsdf.com"
	}]
