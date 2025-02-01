// # 1Example File
//
// First comment is a description of the contents in Markdown format.
// The first heading is the title of the document.
//
// `cue export --out=yaml simple.cue`
//
// more than anything
// 23
//
// 3223
//
// ds f
// sdf
//  sdf sdf
//  sd
// f
//
// sdf
//  sdf
//  sd
// f
// sdf
//  sd
//
// sdf
//  sad
// f
//
// 1. sdf \
// 2. 2 sdfsdf sdf
// 3. sdf sdf sdf
// 4. ds sdfsdfsdf
// 5. sdfsdf s
//
// ## Another heading
//
// --------------------------
//
// and yet again
//
// sdklf jkzasd
// asd
// asdasd

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
		Name:  "First11111"
		Email: "test1@testdomain.com"
	},
	{
		Name: "Second sdk;lfjsdakj skfjlskdjksdjf;sdjf ksdjflk sdfnbsdjfhskjdhf sdjh kjsdh ljsdhksdhf kjlsdhf jhsd ::::::::::::::::::::::::: ::::::::"
		Email: ["test@testdomain.com", "another@sdfklsdjf.com"]
		Notes: """
			sdfksjdflk
			sfsdfsdfasjdflksjdf
			sdfsdf
			ksjd fkjsdfksjd;fljs dkjsdlfkjsdalfsaldf sdifjsdifusadk sdo;fji sdkj sdkfsdlkjskldjslkdjf ;sasad
			11111111111111
			11n1111111111111111n1111111111111111n
			1111111111111111n111111111111
			1111n1111111111111111n11111
			11111111111n11111
			11111111111n1111111
			11111111
			1n1111
			1111
			111111
			11n1111111111111111n1111
			111111111
			111n1111111111111111n11111
			11111111111n11111111111111
			""" @cuebook(detail)
	},
	{
		Name:  "3"
		Email: "test@testdomain.com"
	},
	{
		Name:  "4"
		Email: "test@testdomain.com"
	},
	{
		Name:  "7"
		Email: "test@testdomain.com"
	},
	{
		Name:  "5555555555555555555"
		Email: "test@testdomain.com"
	},
	{
		Name:  "Someone"
		Email: "someEmail@somehost.net"
	},
	{
		Name: """
			8
			888888
			8
			"""
		Email: "test88888@testdomain.com"
	},
	{
		Name:  "10"
		Email: "test@testdomain.com"
		Notes: "sf jsdlfjk sdf sdf kjdsl fjsdlk fjsdlkf jdslkfj sl ksdlfksjdfl ksdjlf jasdklf jsldf jsdlf jsdlkfjlds jflsdjflksjdflkj slfjslfk jdslkf jsdlfjlsfjlksfj lsdjf lksdjf lksjdfl ksdjflk 111"
	},
	{
		Name:  " fjsld fjsld kfjsdl fjsdlk fjsl kfjsld f sdf  sdf kjslk jfsldk fjsdlk fjsdl fkjsdl fjsdlk fjsdl fkjslk fjsdl fkjsdlfkjsdl fkjsdlkfj 6"
		Email: "test@testdomain.com"
	},
	{
		Name:  "Nine is great 99999999999999999999"
		Email: "test@testdomain.com"
	},
	{
		Name:  "Someone1236"
		Email: "someEmail@somehost.net"
	},
	{
		Name:  "11aaaaaz"
		Email: "test@testdomain.com"
	},
	{
		Name:     "Someone0011aa" @cuebook(title)
		Email:    "someEmail12@somehost.net"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$+UbhxEgDdvSIMPTboh8zZA$tcM8iddKv/rBdK8qM45LCAYmPzoVDswiocPITJhfDzA"
	},
	{
		Name:     "Someoneas11"
		Email:    "someEmail@somehost.net"
		Custom:   "xc"
		yayay:    "dsfjsdlkfjk"
		z:        "sdfjdskf klvjclkvxcv"
		Password: "$argon2id$v=19$m=65536,t=3,p=4$I4BZZ90hMBjpc1IoHTN3RQ$KW8S387rBMvBVN4J7KOoHuhdeiE8K54je04cy3Mcayk" @cuebook(detail,trim,argon2id)
	},
  {
		Name:  "Someone!!!!" @cuebook(title)
		Email: "someEmail@somehost.net"
	}
]
