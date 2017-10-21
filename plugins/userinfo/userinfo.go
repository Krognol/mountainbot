package userinfo

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/Krognol/dgofw"
)

var UserinfoHelp = []string{
	"info user -- Displays info about the user for whoever used the command",
	"info user [@mention | id] Displays info about the @'d user.",
	"info avatar -- User's avatar",
	"info server -- Displays server info",
}

var uidregex = regexp.MustCompile("(<@!?[0-9]+>)|([0-9]+)")

func uinfoUser(m *dgofw.DiscordMessage, u *dgofw.DiscordUser) {
	m.ReplyEmbed(&discordgo.MessageEmbed{
		Fields: append(make([]*discordgo.MessageEmbedField, 0),
			&discordgo.MessageEmbedField{
				Name:   "Username",
				Value:  u.Username(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "ID",
				Value:  u.ID(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Discriminator",
				Value:  u.Discriminator(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Registered",
				Value:  u.Timestamp(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Bot",
				Value:  fmt.Sprintf("%t", u.Bot()),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Verified",
				Value:  fmt.Sprintf("%t", u.Verified()),
				Inline: true,
			},
		),
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: u.Avatar(),
		},
		Color: 0xf72e64,
	})
}

func uinfoMember(m *dgofw.DiscordMessage, mem *dgofw.DiscordMember) {
	embed := &discordgo.MessageEmbed{
		Fields: append(make([]*discordgo.MessageEmbedField, 0),
			&discordgo.MessageEmbedField{
				Name:   "Username",
				Value:  mem.User.Username(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "ID",
				Value:  mem.User.ID(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Discriminator",
				Value:  mem.User.Discriminator(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Registered",
				Value:  mem.User.Timestamp(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Bot",
				Value:  fmt.Sprintf("%t", mem.User.Bot()),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Verified",
				Value:  fmt.Sprintf("%t", mem.User.Verified()),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Joined at",
				Value:  mem.JoinedAt(),
				Inline: true,
			},
		),
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: mem.User.Avatar(),
		},
		Color: mem.Color(),
	}

	if mem.Nickname() != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Nickname",
			Value:  mem.Nickname(),
			Inline: true,
		})
	}

	if len(mem.Roles) > 0 {
		result := []string{}
		for _, role := range mem.Roles {
			result = append(result, role.Name)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Roles",
			Value:  strings.Join(result, ",\n"),
			Inline: true,
		})
	}

	m.ReplyEmbed(embed)
}

func serverInfo(m *dgofw.DiscordMessage) {
	g := m.Guild()
	embed := &discordgo.MessageEmbed{
		Title:     "Server info",
		Color:     0xf72e64,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: g.Icon()},
		Image:     &discordgo.MessageEmbedImage{URL: "https://discordapp.com/api/v7/guilds/" + g.ID() + "/embed.png?style=shield"},
		Fields: append(make([]*discordgo.MessageEmbedField, 0),
			&discordgo.MessageEmbedField{
				Name:   "Name",
				Value:  g.Name(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "ID",
				Value:  g.ID(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Region",
				Value:  g.Region(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Owner",
				Value:  g.OwnerID(),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Members",
				Value:  fmt.Sprintf("%d", g.MemberCount()),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Created At",
				Value:  g.CreatedAt(),
				Inline: true,
			},
		),
	}
	if len(g.Emojis()) > 0 {
		var buf bytes.Buffer
		buf.WriteString("Emojis:\n")
		for _, emoji := range g.Emojis() {
			buf.WriteString(fmt.Sprintf("<:%s:%s>", emoji.Name, emoji.ID))
		}
		embed.Description = buf.String()
	}
	m.ReplyEmbed(embed)
}

func UIOnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	switch arg1 {
	case "user":

		var mem *dgofw.DiscordMember
		var usr *dgofw.DiscordUser
		arg2 := m.Arg("arg2")
		if uidregex.MatchString(arg2) {
			var id string
			for _, c := range arg2 {
				if c >= '0' && c <= '9' {
					id += string(c)
				}
			}
			mem = m.Guild().Member(id)
			dgu, err := m.Session().User(id)
			if err != nil {
				m.Reply("Invalid id")
				return
			}
			usr = dgofw.NewDiscordUser(m.Session(), dgu)
		} else {
			mem = m.Guild().Member(m.Author.ID())
		}

		if mem != nil {
			uinfoMember(m, mem)
		} else {
			uinfoUser(m, usr)
		}
	case "server":
		serverInfo(m)
	case "avatar":
		url := m.Author.Avatar()
		res, err := http.Get(url)
		if err != nil {
			m.Reply("Couldn't get avatar")
			return
		}
		var name = m.Author.ID() + "_avatar"
		if strings.HasSuffix(url, ".gif") {
			name += ".gif"
		} else {
			name += ".png"
		}
		m.ReplyFile(name, res.Body)
	}
}
