package memes

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Krognol/dgofw"
	"github.com/jzelinskie/geddit"
)

type Memer struct {
	ses *geddit.Session
}

func NewMemer( /*cid, cs, */ useragent string) *Memer {
	/*oauth, err := geddit.NewOAuthSession(cid, cs, useragent, "")
	oauth.LoginAuth()
	if err != nil {
		fmt.Println(err)
		return nil
	}*/
	return &Memer{
		ses: geddit.NewSession(useragent),
	}
}

func (m *Memer) OnWholesomeMeme(msg *dgofw.DiscordMessage) {
	posts, err := m.ses.SubredditSubmissions("wholesomememes", geddit.HotSubmissions, geddit.ListingOptions{
		Time:  geddit.ThisDay,
		Limit: 25,
	})
	if err != nil {
		msg.Reply("Something happened...")
		fmt.Println(err)
		return
	}

	if len(posts) == 1 {
		msg.Reply(fmt.Sprintf("%s\n%s", posts[0].Title, posts[0].URL))
	}
	rand.Seed(time.Now().UnixNano())
	p := posts[rand.Intn(len(posts))]

	msg.Reply(fmt.Sprintf("%s\n%s", p.Title, p.URL))
}

func (m *Memer) OnDankMeme(msg *dgofw.DiscordMessage) {
	posts, err := m.ses.SubredditSubmissions("dankmemes", geddit.HotSubmissions, geddit.ListingOptions{
		Time:  geddit.ThisDay,
		Limit: 25,
	})
	if err != nil {
		msg.Reply("Something happened...")
		fmt.Println(err)
		return
	}

	if len(posts) == 1 {
		msg.Reply(fmt.Sprintf("%s\n%s", posts[0].Title, posts[0].URL))
	}
	rand.Seed(time.Now().UnixNano())
	p := posts[rand.Intn(len(posts))]

	msg.Reply(fmt.Sprintf("%s\n%s", p.Title, p.URL))
}
