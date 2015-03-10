// Package docparser parses semi-structured documents using regex
//
// A semi-structured document is any text document that doesn't
// have a strict syntax, like YAML, but it's possible to build
// regexes to find specific values from it
//
// The main use for it is to parse automated email messages
package docparser

import (
	"fmt"
	"regexp"
	"strings"
)

// Pattern extracts information from a text
type Pattern interface {
	// Search for the pattern in content
	//
	// Return NoMatch error if pattern not found in content
	Search(content string) (Fields, error)
}

// Fields is the return value of Pattern.Search()
//
// Values could be plain strings or a list of subfields ([]map[string]string)
//
// The functions GetString() and GetMapSlice() handle the type casting
// and return primitive types
type Fields map[string]interface{}

// Update merges other fields into f
//
// If some field from other already exists in f it will be overriden
func (f *Fields) Update(other Fields) {
	for k, v := range other {
		(*f)[k] = v
	}
}

func (f *Fields) Keys() []string {
	keys := make([]string, 0, len(*f))
	for k, _ := range *f {
		keys = append(keys, k)
	}
	return keys
}

// GetString returns the string value associated with key
//
// Return empty string if key is not present or if key
// is present but the value is not a string
func (f *Fields) GetString(key string) (value string) {
	v, ok := (*f)[key]
	if !ok {
		return ""
	}
	vs, ok := v.(string)
	if !ok {
		return ""
	}
	return vs
}

// GetMapSlice return a slice of subfields associated with key
//
// Return empty slice if key is not present or if key
// is present but the value is not a slice of Fields
func (f *Fields) GetMapSlice(key string) []map[string]string {
	v, ok := (*f)[key]
	if !ok {
		return []map[string]string{}
	}
	vf, ok := v.([]Fields)
	if !ok {
		return []map[string]string{}
	}
	vs := make([]map[string]string, len(vf))
	for i, item := range vf {
		vs[i] = make(map[string]string)
		for key, val := range item {
			vs[i][key] = val.(string)
		}
	}
	return vs

}

// NoMatch error returned when Pattern.Search() fails to match
type NoMatch struct {
	Name    string // pattern name that didn't match
	Content string // content the pattern tried to match against
}

func (e *NoMatch) Error() string {
	return fmt.Sprintf("No match for %q", e.Name)
}

// Document is a collection of Patterns
//
// Each Pattern extracts a subset of fields from the content
// and the document fields is the sum of all these individual
// extractions
//
// Document also implements the Pattern interface
type Document []Pattern

func (d *Document) Search(content string) (Fields, error) {
	f := Fields{}
	for _, p := range *d {
		pf, err := p.Search(content)
		if err != nil {
			return Fields{}, err
		}
		f.Update(pf)
	}
	return f, nil
}

// Documents ia a colletion of Document
type Documents []*Document

// Search each Document for content and return the first successfull return
// value
//
// Will try all documents, if all failed return an ErrorList with all errors
func (ds *Documents) Search(content string) (Fields, error) {
	errList := &ErrorList{}
	for i, doc := range *ds {
		fields, err := doc.Search(content)
		if err == nil {
			return fields, nil
		}
		errList.Add(fmt.Errorf("Document %d: %s", i, err.Error()))
	}
	return Fields{}, errList
}

type ErrorList []error

func (el *ErrorList) Add(err error) {
	(*el) = append((*el), err)
}

func (el *ErrorList) Error() string {
	s := make([]string, 0, len(*el))
	for _, err := range *el {
		s = append(s, err.Error())
	}
	return strings.Join(s, "; ")
}

//
// Pattern implementations
//

// PatternGroup ia a Pattern implementation that uses a single regex
// with named groups to extract one or more fields from the content
type PatternGroup struct {
	// Name is a user-friendly identification used for debugging.
	Name string

	// Regex object containing at least one named group.
	Regex *regexp.Regexp

	// Clean is a function that will receive the fields extracted
	// from the regex named groups and should return a cleaned
	// version. Optional.
	Clean func(f Fields) Fields

	// Optional means that if the Regex doesn't match the content
	// given to Search() no error will be returned, just an empty
	// Fields
	//
	// This could simplify the regex
	Optional bool
}

// Search for all named groups from Regex in content
//
// Returns Fields hash where keys are the group names and values
// are the matched values.
//
// Return empty fields and NoMatch error if regex doesn't match
func (pg *PatternGroup) Search(content string) (Fields, error) {
	fields, ok := regexGroups(pg.Regex, content)
	if !ok {
		if pg.Optional {
			return Fields{}, nil
		} else {
			return Fields{}, &NoMatch{pg.Name, content}
		}
	}
	if pg.Clean != nil {
		fields = pg.Clean(fields)
	}
	return fields, nil
}

// PatternList is a Pattern implementation that finds a list of items
// in the content
type PatternList struct {
	Name       string
	ListRegex  *regexp.Regexp
	SplitRegex *regexp.Regexp
	ItemRegex  *regexp.Regexp
	CleanItem  func(f Fields) Fields
	Optional   bool
}

// Search for a list of items in the content using all the regexes
//
// Return value will be a hash with only one key where the value
// is a slice of Fields, i.e.:
//
//    Fields{
//      "items": []Fields{
//        Fields{"key": "item1"},
//        Fields{"key": "item2"},
//        Fields{"key": "item3"},
//      }
//    }
//
func (pl *PatternList) Search(content string) (Fields, error) {
	if !pl.ListRegex.MatchString(content) {
		if pl.Optional {
			return Fields{}, nil
		} else {
			return Fields{}, &NoMatch{pl.Name + " - list regex", content}
		}
	}

	listName := pl.ListRegex.SubexpNames()[1]
	listText := pl.ListRegex.FindStringSubmatch(content)[1]

	itemsTexts := pl.SplitRegex.Split(listText, -1)
	items := []Fields{}

	for i, itemText := range itemsTexts {
		if itemText == "" {
			continue
		}
		fields, ok := regexGroups(pl.ItemRegex, itemText)
		if !ok {
			return Fields{}, &NoMatch{fmt.Sprintf("%s - item %d", pl.Name, i), itemText}
		}
		if pl.CleanItem != nil {
			fields = pl.CleanItem(fields)
		}
		items = append(items, fields)
	}

	return Fields{listName: items}, nil
}

// regexGroups extracts all named groups of the regex re from content
//
// ok will be false if regex doesn't match
func regexGroups(re *regexp.Regexp, content string) (fields Fields, ok bool) {
	if !re.MatchString(content) {
		return Fields{}, false
	}

	fields = Fields{}
	matches := re.FindStringSubmatch(content)

	for i, groupName := range re.SubexpNames() {
		if i == 0 {
			continue // first name is always ""
		}
		fields[groupName] = matches[i]
	}

	return fields, true
}
