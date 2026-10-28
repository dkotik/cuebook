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
		Name:  "sdkjfhsdkjfh12!!as12" @cuebook(title)
		Email: "123aa12312aa@2324.2343"
		Notes: "sdfsfxcvxv"
	},
  {
		Name:  "third and finaleee" @cuebook(title)
		Email: "third@some.email2"
		Notes: "1212a" @cuebook(detail)
	},
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
	},
  {
		Name:  "23424n33" @cuebook(title)
		Email: "234234@31234a1.com"
		Notes: "sdf saf sdfsdf"
	},
  {
		Name:  "123"
		Email: "123@342345.com"
		Notes: "lsdkfjsldfkj"
	},
  {
		Name:     "12sdf33zz"
		Email:    "sdffsd@perfect.com"
		Notes:    "65ggh"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$HIcmGlPcqeg1ARz64fu5Cg$RqThx3got34YXBHH7QyeIQ87Z3wT7CCNwPe/fyZp6cU"
	},
  {
		Name:  "First11111aa1"
		Email: "test1@testdo1main.com"
		Notes: "sd fsdf \nsdf\ns\nadf sdf\nsdf\nsdfs\ndfsdf\n"
	}]
