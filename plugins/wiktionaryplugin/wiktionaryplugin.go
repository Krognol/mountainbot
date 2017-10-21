package wiktionaryplugin

import (
	"fmt"
	"nano/plugins/wiki"

	"github.com/Krognol/dgofw"
	"github.com/bwmarrin/discordgo"
)

func WikiOnMessage(m *dgofw.DiscordMessage) {
	var word string
	if word = m.Arg("word"); word == "" {
		return
	}

	def, pro := wiki.WiktionaryDefinition(word)
	if def == nil {
		m.Reply("Unable to find definition for word '" + word + "'")
		return
	}

	if pro == nil {
		pro = &wiki.WPronounciation{Pr: ""}
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: "https://en.wiktionary.org/favicon.ico",
			Name:    word + " " + pro.Pr,
			URL:     fmt.Sprintf("https://en.wiktionary.com/wiki/%s", word),
		},
		Description: def.Text,
		Footer: &discordgo.MessageEmbedFooter{
			Text: def.Attribution,
		},
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
	}
	m.ReplyEmbed(embed)
}
