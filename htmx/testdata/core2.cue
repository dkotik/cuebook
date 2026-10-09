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
		Name:  "sdkjfhsdkjfh12"
		Email: "12312312@2324.2343"
	},
  {
		Name:  "Adam Bink 2"
		Email: "sdfjkh@sdfs1df.com"
	},
  {
		Name:  "third and finaleee" @cuebook(title)
		Email: "third@some.email2"
		Notes: "1212"
	},
  {
		Name:  "First11111aa1"
		Email: "test1@testdomain.com"
	},
  {
		Name:  "xcxcx" @cuebook(title)
		Email: "cvxvxcv@ds1afsd.com"
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
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	},
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	}]
