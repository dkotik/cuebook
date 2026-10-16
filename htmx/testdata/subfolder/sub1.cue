#email: =~"^[^@]+@[^@]+$"
#contact: {
	// current definition scanner cannot detect abstract definitions yet
	Name: string @cuebook(title)
	Email: #email | [...#email]
	Introduced?: bool
	Notes?:    string @cuebook(detail)
	Password?: string @cuebook(detail, trim, argon2id)
	... // allow any additional fields
}

[...#contact] & [
  {
		Name:       "aFirst11111aaDD222Dzxa11" @cuebook(title)
		Email:      "tes44t1@testdo1m1zzzzain.com"
		Password:   "$argon2id$v=19$m=65536,t=3,p=4$U4eAT5pyeRbmzJq/wd7aew$acreNZzsplw5F9gfGEhU9Mg+cLVuh8k7Y3nKIZFCbBg" @cuebook(detail,trim,argon2id) @cuebook(detail, trim, argon2id)
		Notes:      "sdf sdfsdf12"                                                                                      @cuebook(detail)
		Introduced: true
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
		Name:       "1 really great21" @cuebook(title)
		Email:      "1dfjd@kekeke.kekeke1"
		Notes:      "dfgdf" @cuebook(detail)
		Introduced: true
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
	},
  {
		Name:     "sf"
		Email:    "dsdf@sdf.com"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$cGeXjMyixX+xY8EygAgeOA$dx6GkcrszSRQCO7TA/jhoBzpuj4hdDOWrEWh/w9Rtig"
	},
  {
		Name:     "sdf"
		Email:    "sdf@sdf.com"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$lFPpEgVBiY1UicU755JcKg$j+KEEZIwJPJafycCnfAS035HiGsiuaMns5npt2wZGlI"
	},
  {
		Name:     "ksjdhf" @cuebook(title)
		Email:    "sjdkfh@sdfsdf.com"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$medTMHpZfMcS6rdxNaMa+w$dnueH6PynLQroKz/4jXR9YuxoQtMu7VVjk0fUt+j3iM" @cuebook(detail,trim,argon2id)
		Notes:    "sdfsdf"
	},
  {
		Name:     "23424n33df"
		Email:    "2342sd34@31234a1.com"
		Notes:    "sfs sdf sdfsdf ahem"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$ALV0iRjKqI7BMvHq8Cw46w$2aMZuz76olfzIsIb8/s+9EFTxIXE0tEWBYuqSL70EHU"
	},
  {
		Name:  "new entry"
		Email: "test1@testdomain.com"
		Notes: "sdfsdf"
	}
]
