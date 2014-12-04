package docparser_test

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/RealGeeks/docparser"
)

func ExamplePatternGroup() {
	pattern := &docparser.PatternGroup{
		Name:  "Contact Info",
		Regex: regexp.MustCompile(`Name (?P<name>.*)\nEmail (?P<email>.*)`),
	}

	content := "Name Igor Sobreira\nEmail igor@realgeeks.com"

	fields, err := pattern.Search(content)
	if err != nil {
		panic(err) // check for NoMatch
	}

	fmt.Println(fields.GetString("name"))
	fmt.Println(fields.GetString("email"))
	// Output:
	// Igor Sobreira
	// igor@realgeeks.com
}

func ExamplePatternGroup_clean() {

	pattern := &docparser.PatternGroup{
		Name:  "Name",
		Regex: regexp.MustCompile(`Name (?P<name>.*)\n`),

		// PatternGroup could provide a Clean function to cleanup
		// values from the fields extracted by the regex
		Clean: func(f docparser.Fields) docparser.Fields {
			if name, err := url.QueryUnescape(f.GetString("name")); err == nil {
				f["name"] = name
			}
			return f
		},
	}

	content := "Name Igor%20Sobreira\n"

	fields, err := pattern.Search(content)
	if err != nil {
		panic(err)
	}

	fmt.Println(fields.GetString("name"))
	// Output:
	// Igor Sobreira
}

func ExamplePatternList() {

	pattern := &docparser.PatternList{
		Name: "Languages",
		// ListRegex group 'languages' will have the string
		// that contains all items that will be split later
		//
		// Note the 's' flag to make . (dot) match \n
		ListRegex: regexp.MustCompile(`(?s:Real Geeks languages:\n(?P<languages>.*))`),

		// SplitRegex will use the 'languages' group value and
		// split into multiple strings
		SplitRegex: regexp.MustCompile(`\n`),

		// ItemRegex will use each string from the previous step,
		// extract Fields from this item and add to 'languages'
		// list
		ItemRegex: regexp.MustCompile(` - (?P<name>.*)`),
	}

	content := `Real Geeks languages:
 - Python
 - Ruby
 - Javascript
 - Go
`

	fields, err := pattern.Search(content)
	if err != nil {
		panic(err)
	}

	languages := fields.GetFieldsSlice("languages")

	fmt.Printf("List has %d items\n", len(languages))
	for i, language := range languages {
		fmt.Printf("%d: Language name = %q\n", i, language.GetString("name"))
	}
	// Output:
	// List has 4 items
	// 0: Language name = "Python"
	// 1: Language name = "Ruby"
	// 2: Language name = "Javascript"
	// 3: Language name = "Go"
}

func ExampleDocument() {

	document := &docparser.Document{
		&docparser.PatternGroup{
			Name:  "Contact information",
			Regex: regexp.MustCompile(`Name: (?P<name>.*)\nPhone: (?P<phone>.*)\n`),
		},
		&docparser.PatternList{
			Name:       "Properties viewed",
			ListRegex:  regexp.MustCompile(`(?s:Properties:\n(?P<properties>.*))`),
			SplitRegex: regexp.MustCompile(`\n`),
			ItemRegex:  regexp.MustCompile(` - MLS #(?P<mls>.*) / (?P<address>.*)`),
		},
	}

	content := `Name: Mark Stewart
Phone: (123) 221-1122

Properties:
 - MLS #2211 / 331 Kailua Rd, HI
 - MLS #9090 / 990 Kaelepulu Dr, HI
`

	fields, err := document.Search(content)
	if err != nil {
		panic(err)
	}

	fmt.Println(fields.GetString("name"))
	fmt.Println(fields.GetString("phone"))

	for _, property := range fields.GetFieldsSlice("properties") {
		fmt.Printf("#%s: %s\n", property.GetString("mls"), property.GetString("address"))
	}

	// Output:
	// Mark Stewart
	// (123) 221-1122
	// #2211: 331 Kailua Rd, HI
	// #9090: 990 Kaelepulu Dr, HI

}
