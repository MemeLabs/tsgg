package main

// flair9 - twitchsub
// flair13 - t1
// flair1 - t2
// flair3 - t3
// flair8 - t4
// flair2 - notable
// flair5 - contrib
// protected
// bot
// vip - green/orange
// admin - red

type flair struct {
	Name  string
	Badge string
	Color color
}

// TODO The order here is important for "highestFlair".
// The color which is last in the array will be used for a user with multi-flair.
var legacyflairs = []flair{
	flair{
		Name:  "flair2",
		Badge: "N",
		Color: "",
	},
	flair{
		Name:  "flair5",
		Badge: "C",
		Color: "",
	},
	flair{
		Name:  "flair9",
		Badge: "tw",
		Color: fgBrightBlue,
	},
	flair{
		Name:  "flair13",
		Badge: "t1",
		Color: fgBrightBlue,
	},
	flair{
		Name:  "flair1",
		Badge: "t2",
		Color: fgBrightBlue,
	},
	flair{
		Name:  "flair3",
		Badge: "t3",
		Color: fgBlue,
	},
	flair{
		Name:  "flair8",
		Badge: "t4",
		Color: fgMagenta,
	},
	flair{
		Name:  "flair11",
		Badge: "bot2",
		Color: fgBrightBlack,
	},
	flair{
		Name:  "flair12",
		Badge: "@",
		Color: fgBrightCyan,
	},
	flair{
		Name:  "bot",
		Badge: "bot",
		Color: fgYellow,
	},
	flair{
		Name:  "vip",
		Badge: "vip",
		Color: fgGreen,
	},
	flair{
		Name:  "admin",
		Badge: "@",
		Color: fgRed,
	},
}

var newflairs = []flair{
	flair{
		Name:  "flair2",
		Badge: "N",
		Color: "",
	},
	flair{
		Name:  "flair5",
		Badge: "C",
		Color: "",
	},
	flair{
		Name:  "flair9",
		Badge: "tw",
		Color: fgBrightBlue,
	},
	flair{
		Name:  "flair13",
		Badge: "t1",
		Color: fgBrightBlue,
	},
	flair{
		Name:  "flair1",
		Badge: "t2",
		Color: fgBrightCyan,
	},
	flair{
		Name:  "flair3",
		Badge: "t3",
		Color: fgGreen,
	},
	flair{
		Name:  "flair8",
		Badge: "t4",
		Color: fgMagenta,
	},
	flair{
		Name:  "flair11",
		Badge: "bot2",
		Color: fgBrightBlack,
	},
	flair{
		Name:  "flair12",
		Badge: "@",
		Color: fgBrightCyan,
	},
	flair{
		Name:  "bot",
		Badge: "bot",
		Color: fgBlue,
	},
	flair{
		Name:  "vip",
		Badge: "vip",
		Color: fgBrightRed,
	},
	flair{
		Name:  "admin",
		Badge: "@",
		Color: fgRed,
	},
}
