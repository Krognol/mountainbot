package tags

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/Krognol/dgofw"
)

type Tag struct {
	Name    string
	OwnerID string `json:"owner_id"`
	Content string `json:"content"`
}

type Server struct {
	ID   string `json:"id"`
	Tags []*Tag `json:"tags"`
}

type Tags struct {
	sync.RWMutex
	Guilds []*Server `json:"servers"`
}

var TagsHelp = []string{
	"tag [name]                -- Sends the contents of the tag.",
	"tag add [name] [content]  -- Adds a new tag.",
	"tag remove [name]         -- Removes the tag, only usable by mods and the tag owner of the tag.",
	"tag edit [name] [content] -- Edits a tag, only usable by mods and the tag owner.",
}

func (t *Tags) load() {
	if b, err := ioutil.ReadFile("./tagsstate.json"); err == nil {
		json.Unmarshal(b, t)
	}
}

func NewTags() *Tags {
	result := &Tags{
		Guilds: []*Server{},
	}

	result.load()
	return result
}

func (t *Tags) save() {
	if f, err := os.Create("./tagsstate.json"); err == nil {
		b, _ := json.Marshal(t)
		f.Write(b)
	}
}

func (t *Tags) addTag(m *dgofw.DiscordMessage, name, content string) {
	t.Lock()
	defer t.Unlock()
	tg := m.GuildID()
	for _, guild := range t.Guilds {
		if guild.ID == tg {
			guild.Tags = append(guild.Tags, &Tag{
				OwnerID: m.Author.ID(),
				Content: content,
				Name:    name,
			})
			m.Reply("'Added tag '" + name + "'")
			return
		}
	}
}

func (t *Tags) delTag(m *dgofw.DiscordMessage, name string) {
	tg := m.GuildID()
	for _, guild := range t.Guilds {
		if guild.ID == tg {
			for i, tag := range guild.Tags {
				if tag.Name == name {
					if m.IsMod() || m.Author.ID() == tag.OwnerID {
						t.Lock()
						guild.Tags = append(guild.Tags[:i-1], guild.Tags[i:]...)
						t.Unlock()
						m.Reply("Removed tag '" + name + "'")
						return
					}
				}
			}
		}
	}
}

func (t *Tags) editTag(m *dgofw.DiscordMessage, name, content string) {
	tg := m.GuildID()
	for _, guild := range t.Guilds {
		if guild.ID == tg {
			for _, tag := range guild.Tags {
				if tag.Name == name {
					if m.IsMod() || m.Author.ID() == tag.OwnerID {
						t.Lock()
						tag.Content = content
						t.Unlock()
						m.Reply("Edited tag '" + name + "'")
						return
					}
				}
			}
		}
	}
}

func (t *Tags) getTag(m *dgofw.DiscordMessage, name string) {
	tg := m.GuildID()
	for _, guild := range t.Guilds {
		if guild.ID == tg {
			for _, tag := range guild.Tags {
				if tag.Name == name {
					m.Reply(tag.Content)
					return
				}
			}
			var buf bytes.Buffer
			buf.WriteString("Couldn't find tag '" + name + "'.\n")
			for _, tag := range guild.Tags {
				if strings.Contains(tag.Name, name) {
					buf.WriteString("**" + tag.Name + "**\n")
				}
			}
			if len(guild.Tags) > 0 {
				m.Reply("Did you mean:\n" + buf.String())
			} else {
				m.Reply("Couldn't find tag '" + name + "'.")
			}
			return
		}
	}
}

func (t *Tags) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	switch arg1 {
	case "add":
		name, content := m.Arg("arg2"), m.Arg("arg3")
		if name == "" || content == "" {
			return
		}

		t.addTag(m, name, content)
	case "remove":
		name := m.Arg("arg2")
		if name == "" {
			return
		}
		t.delTag(m, name)
	case "edit":
		name, content := m.Arg("arg2"), m.Arg("arg3")
		if name == "" || content == "" {
			return
		}

		t.editTag(m, name, content)
	default:
		name := m.Arg("arg1")
		t.getTag(m, name)
	}
}
