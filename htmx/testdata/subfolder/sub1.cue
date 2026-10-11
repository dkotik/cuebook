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
		Name:     "aFirst11111aaDDD" @cuebook(title)
		Email:    "test1@testdo1main.com"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$U4eAT5pyeRbmzJq/wd7aew$acreNZzsplw5F9gfGEhU9Mg+cLVuh8k7Y3nKIZFCbBg" @cuebook(detail,trim,argon2id)
	},
  {
		Name:     "12sdf33zz" @cuebook(title)
		Email:    "sdffsd@perfect.com"
		Notes:    "65ggh" @cuebook(detail)
		Password: "$argon2id$v=19$m=65536,t=3,p=4$HIcmGlPcqeg1ARz64fu5Cg$RqThx3got34YXBHH7QyeIQ87Z3wT7CCNwPe/fyZp6cU"
	},
  {
		Name:  "2378" @cuebook(title)
		Email: "sdf@sdf.com1"
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
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	},
  {
		Name:  "Super Rambo 123"
		Email: "test1@testdom1ain.com"
	},
  {
		Name:  "Super Rambo 123" @cuebook(title)
		Email: "test1@testdom1ain.com"
	},
  {
		Name:  "Super Rambo 123" @cuebook(title)
		Email: "test1@tes333333tdom12ain.com"
	},
  {
		Name:  "Super Rambo 123" @cuebook(title)
		Email: "test1@testdom1ain.com"
	}
]
