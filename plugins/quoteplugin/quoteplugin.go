package quoteplugin

import (
	"io/ioutil"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"fmt"

	"os"

	"encoding/json"

	"github.com/Krognol/dgofw"
	"github.com/google/go-github/github"
)

type Server struct {
	ID     string   `json:"id"`
	Quotes []string `json:"quotes"`
}

type Quotes struct {
	sync.RWMutex
	Servers []*Server `json:"servers"`
}

func NewQuotePlugin() *Quotes {
	plugin := &Quotes{Servers: make([]*Server, 0)}
	err := plugin.Load()
	if err != nil {
		fmt.Println(err.Error())
		plugin.Save()
	}
	return plugin
}

var QuotesHelp = []string{
	"quote  -- returns a random quote",
	"quote [index] -- returns a quote",
	"quote add [quote] -- adds a quote",
	"quote del [id] -- deletes a quote with the given id, e.g. 163",
	"quote list -- creates a gist with every quote",
}

func (q *Quotes) getServer(id string) *Server {
	for _, s := range q.Servers {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (q *Quotes) addQuote(m *dgofw.DiscordMessage) {
	if quote := m.Arg("arg2"); quote != "" {
		s := q.getServer(m.GuildID())
		if s == nil {
			s = &Server{
				ID:     m.GuildID(),
				Quotes: make([]string, 0),
			}
		}
		q.Lock()
		s.Quotes = append(s.Quotes, quote)
		q.Servers = append(q.Servers, s)
		q.Unlock()
		q.Save()
		m.Reply(fmt.Sprintf("Added quote #%d", len(s.Quotes)))
	}
}

func (q *Quotes) delQuote(m *dgofw.DiscordMessage, index int) {
	if s := q.getServer(m.GuildID()); s != nil {
		if index > 0 && index < len(s.Quotes) {
			if len(s.Quotes) == 1 {
				s.Quotes = s.Quotes[1:]
				return
			}

			q.Lock()
			s.Quotes = append(s.Quotes[:index], s.Quotes[index+1:]...)
			q.Unlock()
			q.Save()
		}
	}
}

func (q *Quotes) sendList(m *dgofw.DiscordMessage) {
	if s := q.getServer(m.GuildID()); s != nil {
		content := strings.Join(s.Quotes, "\n")
		gc := github.NewClient(nil)

		id := "quotes"
		pub := true
		files := make(map[github.GistFilename]github.GistFile)
		files[github.GistFilename("quotes.txt")] = github.GistFile{
			Content: &content,
		}
		gist := &github.Gist{
			ID:     &id,
			Public: &pub,
			Files:  files,
		}
		g, _, err := gc.Gists.Create(nil, gist)
		if err != nil {
			m.Reply("Failed to create gist.\n" + err.Error())
			return
		}
		m.Reply(*g.HTMLURL)
	}
}

func (q *Quotes) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	switch arg1 {
	case "add":
		q.addQuote(m)
	case "del":
		index := m.Arg("arg2")
		if i, err := strconv.ParseInt(index, 10, 64); err == nil {
			q.delQuote(m, int(i))
		} else {
			m.Reply("Invalid number")
		}
	case "list":
		q.sendList(m)
	case "":
		if s := q.getServer(m.GuildID()); s != nil {
			index := rand.Intn(len(s.Quotes))
			m.Reply(fmt.Sprintf("%d. %s", index, s.Quotes[index]))
		} else {
			m.Reply("There are no quotes!")
		}
	default:
		if i, err := strconv.ParseInt(m.Arg("arg2"), 10, 64); err == nil {
			if s := q.getServer(m.GuildID()); s != nil {
				if i > 0 && int(i) < len(s.Quotes) {
					m.Reply(s.Quotes[i])
				}
			}
		}
	}
}

func (q *Quotes) Save() (err error) {
	var f *os.File
	defer f.Close()
	if f, err = os.Create("./quotesstate.json"); err == nil {
		return json.NewEncoder(f).Encode(q)
	}
	return
}

func (p *Quotes) Load() (err error) {
	var b []byte
	if b, err = ioutil.ReadFile("./quotesstate.json"); err == nil {
		return json.Unmarshal(b, p)
	}
	return
}
