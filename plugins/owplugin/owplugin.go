package owplugin

import (
	// Please excuse this
	"nano/plugins/ow"

	"github.com/Krognol/dgofw"
)

func OWOnMessage(m *dgofw.DiscordMessage) {
	bt := m.Arg("battletag")
	region := m.Arg("region")
	if bt == "" || region == "" {
		return
	}

	stats, err := ow.GetStats(bt, region)
	if err != nil {
		m.Reply("Something happened...")
		return
	}

	m.ReplyEmbed(stats)
}
