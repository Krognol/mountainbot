package udplugin

import (
	"nano/plugins/ud"

	"github.com/Krognol/dgofw"
	"github.com/bwmarrin/discordgo"
)

func UrbanOnMessage(m *dgofw.DiscordMessage) {
	var wword string
	if wword = m.Arg("word"); wword == "" {
		return
	}
	ud := ud.GetUDDefiniton(wword)
	if ud == nil || len(ud.List) == 0 {
		m.Reply("Couldn't find a definition for '" + wword + "'")
		return
	}
	word := ud.List[0]

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			URL:     word.URL,
			IconURL: "http://www.urbandictionary.com/favicon.ico",
			Name:    wword,
		},
		Description: word.Definition + "\n\n" + word.Example,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "from UrbanDictionary",
		},
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
	}

	m.ReplyEmbed(embed)
}
