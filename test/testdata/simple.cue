// cue export --out=cue simple.cue

#email: =~ "^[^@]+@[^@]+$"

[...{
	Name:   string
	Email:  #email | [...#email]
	Notes?: string
}] & [
  {
  	Name: "First"
  	Email: "test@testdomain.com"
  },
  {
  	Name: "Second"
  	Email: [
      "test@testdomain.com",
      "another@sdfklsdjf.com"
    ]
    Notes: "sdfksjdflk\nsfsdfsdfasjdflksjdf\nsdfsdf"
  },
]
