package opeth

import (
	"math/rand"

	"io/ioutil"
	"strings"

	"github.com/Krognol/dgofw"
)

type Opeth struct {
	lines []string
	g     *Generator
}

func NewOpethPlugin() *Opeth {
	b, err := ioutil.ReadFile("./opeth_record.txt")
	if err != nil {
		return nil
	}
	plugin := &Opeth{lines: strings.Split(string(b), "\n"), g: CreateGenerator(1, 500)}
	for _, line := range plugin.lines {
		plugin.g.AddSeeds(line)
	}
	return plugin
}

func (o *Opeth) OnMessage(m *dgofw.DiscordMessage) {
	m.Reply(o.g.GenerateText())
}

// Since both maps (the prefix -> suffix and canonical -> representation)
// operate about the same way, we abstract their representation into a notion
// of CountedStrings, where the values of the map contain both the string we
// care about and a count of how often it occurs.
type CountedString struct {
	hits int
	str  string
}

// A CountedStringList is a list of all the CountedStrings for a given prefix,
// and a total number of times that prefix occurs (necessary, with the
// CountedString hits, for probability calculation).
type CountedStringList struct {
	slice []*CountedString
	total int
}

// Map from a prefix in canonical form to CountedStringLists, where one will
// move canonical prefixes to suffixes, and another to words -> representation.
type CountedStringMap map[string]*CountedStringList

// Generators gives us all we need to build a fresh data model to generate
// from.
type Generator struct {
	PrefixLen  int
	CharLimit  int
	Data       CountedStringMap // suffix map
	Reps       CountedStringMap // representation map
	Beginnings []string         // acceptable ways to start a tweet.
}

// CreateGenerator returns a Generator that is fully initialized and ready for
// use.
func CreateGenerator(prefixLen int, charLimit int) *Generator {
	markov := make(CountedStringMap)
	reps := make(CountedStringMap)
	beginnings := []string{}
	return &Generator{prefixLen, charLimit, markov, reps, beginnings}
}

// Convenience method, already populating the first "hit" of the CountedString.
func createCountedString(str string) *CountedString {
	return &CountedString{1, str}
}

// AddSeeds takes in a string, breaks it into prefixes, and adds it to the
// data model.
func (g *Generator) AddSeeds(input string) {
	source := tokenize(input)

	first := true
	for len(source) > g.PrefixLen {
		prefix := strings.Join(source[0:g.PrefixLen], " ")
		AddToMap(prefix, source[g.PrefixLen], g.Data)
		source = source[1:]
		if first {
			g.Beginnings = append(g.Beginnings, prefix)
			first = false
		}
	}
}

// Add to map checks if the key/value pair exists in the map. If not, we create
// them, and if so, we either increment the counter on the value or initialize
// it if it didn't exist previously.
func AddToMap(prefix, toAdd string, aMap CountedStringMap) {
	if csList, exists := aMap[prefix]; exists {
		if countedStr, member := csList.hasCountedString(toAdd); member {
			countedStr.hits++
		} else {
			countedStr = createCountedString(toAdd)
			csList.slice = append(csList.slice, countedStr)
		}
		csList.total++
	} else {
		countedStr := createCountedString(toAdd)
		countedStrSlice := make([]*CountedString, 0)
		countedStrSlice = append(countedStrSlice, countedStr)
		csList := &CountedStringList{countedStrSlice, 1}

		aMap[prefix] = csList
	}
}

// tokenize splits the input string into "words" we use as prefixes and
// suffixes. We can't do a naive 'split' by a separator, or even a regex '\W'
// due to corner cases, and the nature of the text we intend to capture: e.g.
// we'd like "forty5" to parse as such, rather than "forty" with "5" being
// interpreted as a "non-word" character. Similarly with hashtags, etc.
func tokenize(input string) []string {
	return strings.Split(input, " ")
}

// hasCountedString searches a CountedStringList for one that contains the string, and
// returns the suffix (if applicable) and a boolean describing whether or not
// we found it.
func (l CountedStringList) hasCountedString(lookFor string) (*CountedString, bool) {
	slice := l.slice
	for i := 0; i < len(slice); i++ {
		curr := slice[i]
		if curr.str == lookFor {
			return curr, true
		}
	}

	return createCountedString(""), false
}

// Generates text from the given generator. It stops when the character limit
// has run out, or it encounters a prefix it has no suffixes for.
func (g *Generator) GenerateText() string {
	return g.GenerateFromPrefix(g.randomPrefix())
}

// We expose this version primarily for testing.
func (g *Generator) GenerateFromPrefix(prefix string) string {

	// Representation gets a special case, since you can have a multi-word
	// prefix (e.g. "Paul is") but each word needs it's own representation
	// (e.g. "PAUL" "is" or "pAUL" "Is"). Note that this can break if your
	// prefix's rep is longer than the charLimit, should we generalize
	var result []string
	charLimit := g.CharLimit

	result = append(result, prefix)
	charLimit -= len(prefix)

	for {
		word, shouldTerminate, newPrefix, newCharLimit := g.popNextWord(prefix, charLimit)
		prefix = newPrefix
		charLimit = newCharLimit

		if shouldTerminate {
			break
		} else {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

func (g *Generator) popNextWord(prefix string, limit int) (string, bool, string, int) {

	csList, exists := g.Data[prefix]

	if !exists {
		return "", true, "", 0 // terminate path
	}
	successor := csList.DrawProbabilistically()
	var rep string

	rep = successor

	addsTo := len(rep) + 1

	if addsTo <= limit {
		shifted := append(strings.Split(prefix, " ")[1:], rep)
		newPrefix := strings.Join(shifted, " ")
		newLimit := limit - addsTo
		return rep, false, newPrefix, newLimit
	}

	return "", true, "", 0
}

func (cs CountedStringList) DrawProbabilistically() string {
	index := rand.Intn(cs.total) + 1
	for i := 0; i < len(cs.slice); i++ {
		if index <= cs.slice[i].hits {
			return cs.slice[i].str
		}
		index -= cs.slice[i].hits
	}
	return ""
}

func (g *Generator) randomPrefix() string {
	index := rand.Intn(len(g.Beginnings))
	return g.Beginnings[index]
}

// For testing.
func (s *CountedStringList) GetSuffix(lookFor string) (*CountedString, bool) {
	for i := 0; i < len(s.slice); i++ {
		if s.slice[i].str == lookFor {
			return s.slice[i], true
		}
	}
	return createCountedString(""), false
}
