package gfycat

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/Krognol/dgofw"
	"github.com/Krognol/gofycat"
	"github.com/bwmarrin/discordgo"
)

type GfyCatPlugin struct {
	client *gofycat.Cat
}

func NewGfyCatPlugin(id, secret string, typ gofycat.GrantType) *GfyCatPlugin {
	return &GfyCatPlugin{
		client: gofycat.New(id, secret, typ),
	}
}

var GfyHelp = []string{
	"gfy [text] -- searches for a gfycat with the relevant text",
	"gfy trending -- returns a list of trending gfy tags",
	"gfy trending [tag] -- returns a trending gfy with the supplied tag",
	"gfy user [username] -- search for a gfycat user",
}

func (g *GfyCatPlugin) handleTrendingWithTag(m *dgofw.DiscordMessage, tag string) {
	gifs, err := g.client.GetTrendingGfycats(tag, "")
	if err != nil {
		m.Reply("Couldn't find a trending gfy with that tag.")
		return
	}

	for _, gif := range gifs {
		if i, ok := gif.NSFW.(int); ok && i <= 1 {
			m.ReplyEmbed(&discordgo.MessageEmbed{
				Image: &discordgo.MessageEmbedImage{
					URL: gif.GifURL,
				},
			})
			return
		} else if s, ok := gif.NSFW.(string); ok && s != "" {
			if s == "0" || s == "1" {
				m.ReplyEmbed(&discordgo.MessageEmbed{
					Image: &discordgo.MessageEmbedImage{
						URL: gif.GifURL,
					},
				})
				return
			}
		}
	}
	m.Reply("Couldn't find an appropriate gif")
}

func (g *GfyCatPlugin) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	switch arg1 {
	case "trending":
		if arg2 := m.Arg("arg2"); arg2 != "" {
			g.handleTrendingWithTag(m, arg2)
		} else {
			trend, err := g.client.GetTrendingTags()
			if err != nil {
				m.Reply("Something happened...")
				return
			}
			m.Reply("**Trending Gfycat tags**:\n" + strings.Join(trend, "\n"))
		}
	case "user":
		if u := m.Arg("arg2"); u != "" {
			user, err := g.client.GetUser(u)
			if err != nil {
				m.Reply("Something happened...")
				return
			}

			embed := &discordgo.MessageEmbed{
				Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
				Author: &discordgo.MessageEmbedAuthor{
					URL:     user.ProfileURL,
					Name:    user.Username,
					IconURL: user.ProfileImageURL,
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: user.ProfileImageURL,
				},
				Fields: append(make([]*discordgo.MessageEmbedField, 0),
					&discordgo.MessageEmbedField{
						Name:   "User ID",
						Value:  user.Userid,
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Created at",
						Value:  user.CreateDate,
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Followers",
						Value:  user.Followers,
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Following",
						Value:  user.Following,
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Verified",
						Value:  fmt.Sprintf("%t", user.Verified),
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Published Gfys",
						Value:  user.PublishedGfycats,
						Inline: true,
					}, &discordgo.MessageEmbedField{
						Name:   "Published Albums",
						Value:  user.PublishedAlbums,
						Inline: true,
					}),
			}

			m.ReplyEmbed(embed)
		}
	default:
		if arg2 := m.Arg("arg2"); arg2 != "" {
			gfys, err := g.client.SearchGfycats(arg2)
			if err != nil {
				m.Reply("Something happened...")
				return
			}
			i := rand.Intn(len(gfys.Gfycats))
			gfy := gfys.Gfycats[i]
			m.ReplyEmbed(&discordgo.MessageEmbed{
				Image: &discordgo.MessageEmbedImage{
					URL: gfy.GifURL,
				},
			})
		}
	}
}
