// # Example File
//
// First comment is a description of the contents in Markdown format.
// The first heading is the title of the document.
//
// `cue export --out=yaml simple.cue`

#email: =~"^[^@]+@[^@]+$"
#contact: {
	Name: string
	Email: #email | [...#email]
	Notes?: string
	... // allow any additional fields
}

[...#contact] & [
	{
		Name:  "First12za"
		Email: "test@testdomain.com"
	},
	{
		Name: "Second sdk;lfjsdakj skfjlskdjksdjf;sdjf ksdjflk sdfnbsdjfhskjdhf sdjh kjsdh ljsdhksdhf kjlsdhf jhsd ::::::::::::::::::::::::: ::::::::"
		Email: [
			"test@testdomain.com",
			"another@sdfklsdjf.com",
		]
		Notess: "sdfksjdflk\nsfsdfsdfasjdflksjdf\nsdfsdf\nksjd fkjsdfksjd;fljs dkjsdlfkjsdalfsaldf sdifjsdifusadk sdo;fji sdkj sdkfsdlkjskldjslkdjf ;sasad\n11111111111111\n11n1111111111111111n1111111111111111n\n1111111111111111n111111111111\n1111n1111111111111111n11111\n11111111111n11111\n11111111111n1111111\n11111111\n1n1111\n1111\n111111\n11n1111111111111111n1111\n111111111\n111n1111111111111111n11111\n11111111111n1111111111111111n11111\n1111111111\n\\n1111111111n111111111\n11111111\n111111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n1111111111\n111111111111"
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
		Name:  "5555555555555555555"
		Email: "test@testdomain.com"
	},
	{
		Name:  "7"
		Email: "test@testdomain.com"
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
		Name:  "Nine is great 99999999999999999999"
		Email: "test@testdomain.com"
	},
	{
		Name:  "10"
		Email: "test@testdomain.com"
		Notes: "sf jsdlfjk sdf sdf kjdsl fjsdlk fjsdlkf jdslkfj sl ksdlfksjdfl ksdjlf jasdklf jsldf jsdlf jsdlkfjlds jflsdjflksjdflkj slfjslfk jdslkf jsdlfjlsfjlksfj lsdjf lksdjf lksjdfl ksdjflk 111"
	},
	{
		Name:  "11aaaaaz"
		Email: "test@testdomain.com"
	},
	{
		Name:  " fjsld fjsld kfjsdl fjsdlk fjsl kfjsld f sdf  sdf kjslk jfsldk fjsdlk fjsdl fkjsdl fjsdlk fjsdl fkjslk fjsdl fkjsdlfkjsdl fkjsdlkfj 6"
		Email: "test@testdomain.com"
	},
	{
		Name:  "Someone"
		Email: "someEmail@somehost.net"
	},
  {
		Name:  "Someone"
		Email: "someEmail@somehost.net"
	},
          
            
    
                
        
    
        
      
        
        
  
  
      
  
								]
