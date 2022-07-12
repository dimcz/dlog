package filters

import (
	"bufio"
	"errors"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/runes"
	"github.com/dimcz/dlog/utils"

	"github.com/nsf/termbox-go"
)

type FilterResult uint8

const FilterMinLength int = 2

//goland:noinspection GoUnusedConst
const (
	FilterNoaction FilterResult = iota
	FilterIncluded
	FilterExcluded
	FilterHighlighted
)

//goland:noinspection GoUnusedType
type filter interface {
	takeAction(str []rune, currentAction FilterResult) FilterResult
}

type SearchType struct {
	ID    uint8 // id will be generated by order defined in init
	Color termbox.Attribute
	Name  string
}

var CaseSensitive = SearchType{
	Color: termbox.ColorYellow,
	Name:  "CaseS",
}

var RegEx = SearchType{
	Color: termbox.ColorRed,
	Name:  "RegEx",
}
var SearchTypeMap map[uint8]SearchType

type FilterAction uint8

const (
	FilterIntersect FilterAction = iota
	FilterUnion
	FilterExclude
	FilterHighlight
)

const (
	FilterIntersectChar rune = '&'
	FilterUnionChar     rune = '+'
	FilterExcludeChar   rune = '-'
	FilterHighlightChar rune = '~'
)

var FilterActionMap = map[rune]FilterAction{
	FilterIntersectChar: FilterIntersect,
	FilterUnionChar:     FilterUnion,
	FilterExcludeChar:   FilterExclude,
	FilterHighlightChar: FilterHighlight,
}

func init() {
	SearchTypeMap = make(map[uint8]SearchType)
	// Should maintain order, otherwise history will be corrupted.
	for i, r := range []*SearchType{&CaseSensitive, &RegEx} {
		r.ID = uint8(i)
		SearchTypeMap[r.ID] = *r
	}

}

// SearchFunc Follows regex return value pattern. nil if not found, slice of range if found
// Filter does not really need it, but highlighting also must search and requires it
type SearchFunc func(sub []rune) []int
type ActionFunc func(str []rune, currentAction FilterResult) FilterResult

type Filter struct {
	sub        []rune
	st         SearchType
	Action     FilterAction
	TakeAction ActionFunc
}

var ErrBadFilterDefinition = errors.New("bad filter definition")

func NewFilter(sub []rune, action FilterAction, searchType SearchType) (*Filter, error) {
	ff, err := GetSearchFunc(searchType, sub)
	if err != nil {
		return nil, err
	}
	var af ActionFunc
	switch action {
	case FilterIntersect:
		af = buildIntersectionFunc(ff)
	case FilterUnion:
		af = buildUnionFunc(ff)
	case FilterExclude:
		af = buildExcludeFunc(ff)
	case FilterHighlight:
		af = buildHighlightFunc(ff)
	default:
		return nil, ErrBadFilterDefinition
	}

	return &Filter{
		sub:        sub,
		st:         searchType,
		Action:     action,
		TakeAction: af,
	}, nil
}

func GetSearchFunc(searchType SearchType, sub []rune) (SearchFunc, error) {
	var ff SearchFunc
	switch searchType {
	case CaseSensitive:
		subLen := len(sub)
		ff = func(str []rune) []int {
			i := runes.Index(str, sub)
			if i == -1 {
				return nil
			}
			return []int{i, i + subLen}
		}
	case RegEx:
		re, err := regexp.Compile(string(sub))
		if err != nil {
			return nil, ErrBadFilterDefinition
		}
		ff = func(str []rune) []int {
			return re.FindStringIndex(string(str))
		}
	default:
		return nil, ErrBadFilterDefinition
	}

	return ff, nil
}

func IndexAll(searchFunc SearchFunc, runestack []rune) (indices [][]int) {
	if len(runestack) == 0 {
		return
	}
	var i int
	var ret []int
	f := 0
	indices = make([][]int, 0, 1)
	for {
		ret = searchFunc(runestack[i:])
		f++
		if ret == nil {
			break
		} else {
			ret[0] = ret[0] + i
			ret[1] = ret[1] + i
			indices = append(indices, ret)
			i = i + ret[1]
		}
		if i >= len(runestack) {
			break
		}
	}

	return
}

func buildUnionFunc(searchFunc SearchFunc) ActionFunc {
	return func(str []rune, currentAction FilterResult) FilterResult {
		if currentAction == FilterHighlighted {
			return FilterHighlighted
		}
		if currentAction == FilterIncluded {
			return FilterIncluded
		}
		if searchFunc(str) != nil {
			return FilterIncluded
		}

		return FilterExcluded
	}

}

func buildIntersectionFunc(searchFunc SearchFunc) ActionFunc {
	return func(str []rune, currentAction FilterResult) FilterResult {
		if currentAction == FilterHighlighted {
			return FilterHighlighted
		}
		if currentAction == FilterExcluded {
			return FilterExcluded
		}
		if searchFunc(str) != nil {
			return FilterIncluded
		}

		return FilterExcluded
	}
}

func buildExcludeFunc(searchFunc SearchFunc) ActionFunc {
	return func(str []rune, currentAction FilterResult) FilterResult {
		if currentAction == FilterHighlighted {
			return FilterHighlighted
		}
		if currentAction == FilterExcluded {
			return FilterExcluded
		}
		if searchFunc(str) != nil {
			return FilterExcluded
		}

		return FilterIncluded
	}
}

func buildHighlightFunc(searchFunc SearchFunc) ActionFunc {
	return func(str []rune, currentAction FilterResult) FilterResult {
		if currentAction == FilterHighlighted {
			return FilterHighlighted
		}
		if searchFunc(str) != nil {
			return FilterHighlighted
		}

		return currentAction
	}
}

func getFilterAction(filterStr []rune) (FilterAction, error) {
	sign := filterStr[0]
	action, ok := FilterActionMap[sign]

	if !ok {
		return action, &UnknownFilterTypeError{string(sign), ""}
	}

	if len(filterStr) < FilterMinLength {
		return action, &FilterTooShortError{string(filterStr), ""}
	}

	return action, nil
}

func parseFilterLine(line string) (*Filter, error) {
	trimmedLine := []rune(strings.TrimLeftFunc(line, unicode.IsSpace))
	if len(trimmedLine) == 0 {
		return nil, nil
	}
	action, err := getFilterAction(trimmedLine)
	if err != nil {
		return nil, err
	}
	filter, err := NewFilter(
		trimmedLine[1:],
		action,
		CaseSensitive,
	)

	if err != nil {
		return nil, err
	}

	return filter, nil
}

func ParseFiltersFile(filename string) ([]*Filter, error) {
	if err := utils.ValidateRegularFile(utils.ExpandHomePath(filename)); err != nil {
		return nil, err
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer logging.LogOnErr(f.Close())
	scanner := bufio.NewScanner(f)

	var filters []*Filter
	for scanner.Scan() {
		filter, err := parseFilterLine(scanner.Text())
		if err != nil {
			if er, ok := err.(*UnknownFilterTypeError); ok {
				er.Filename = filename
			} else if er, ok := err.(*FilterTooShortError); ok {
				er.Filename = filename
			}
			return nil, err
		}
		if filter == nil {
			continue
		}
		filters = append(filters, filter)
	}

	return filters, nil
}

//goland:noinspection GoUnusedExportedFunction
func ParseFiltersOpt(optStr string) ([]*Filter, error) {
	re := regexp.MustCompile("([^;]+);?")
	var filters []*Filter
	for _, m := range re.FindAllStringSubmatch(optStr, -1) {
		if strings.TrimFunc(m[1], unicode.IsSpace) == "" {
			continue
		}
		filter, err := parseFilterLine(m[1])
		if filter != nil {
			filters = append(filters, filter)
			continue
		} else if err != nil {
			switch err.(type) {
			case *FilterTooShortError:
				return nil, err
			default:
			}
		}
		fileFilters, err := ParseFiltersFile(m[1])
		if err != nil {
			return nil, err
		}
		filters = append(filters, fileFilters...)
	}

	return filters, nil
}
