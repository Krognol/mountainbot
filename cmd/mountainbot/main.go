package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"nano/plugins/lenny"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/Krognol/dgofw"
	"github.com/Krognol/gofycat"
	"github.com/Krognol/mountainbot/plugins/gfycat"
	"github.com/Krognol/mountainbot/plugins/lastfm"
	"github.com/Krognol/mountainbot/plugins/malist"
	"github.com/Krognol/mountainbot/plugins/memes"
	"github.com/Krognol/mountainbot/plugins/music"
	"github.com/Krognol/mountainbot/plugins/opeth"
	"github.com/Krognol/mountainbot/plugins/owplugin"
	"github.com/Krognol/mountainbot/plugins/quoteplugin"
	"github.com/Krognol/mountainbot/plugins/spotifyplugin"
	"github.com/Krognol/mountainbot/plugins/tags"
	"github.com/Krognol/mountainbot/plugins/udplugin"
	"github.com/Krognol/mountainbot/plugins/userinfo"
	"github.com/Krognol/mountainbot/plugins/wiktionaryplugin"
	"github.com/Krognol/mountainbot/plugins/wolframplugin"
)

type (
	// IDSecretPair is a pair of a web app ``client_id`` and ``client_secret``
	IDSecretPair struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	// Unused for now
	Options struct {
		NSFW    bool              `json:"nsfw"`
		Modules map[string]string `json:"modules"`
	}
	Server struct {
		ID      string   `json:"id"`
		Options *Options `json:"options"`
	}

	// Modules is a collection of modules
	Config struct {
		sync.RWMutex
		Servers []*Server `json:"servers"`
		Modules struct {
			Discord struct {
				Token  string `json:"token"`
				Prefix string `json:"prefix"`
			} `json:"discord"`
			Gfycat IDSecretPair `json:"gfycat"`
			LastFM struct {
				AppID string `json:"appid"`
			} `json:"lastfm"`
			Weebery struct {
				MAL struct {
					Username string `json:"username"`
					Password string `json:"password"`
				} `json:"mal"`
				Anilist IDSecretPair `json:"anilist"`
			} `json:"weebery"`
			Wolfram struct {
				AppID string `json:"appid"`
			} `json:"wolfram"`
			Spotify IDSecretPair `json:"spotify"`
			Reddit  IDSecretPair `json:"reddit"`
			Logging struct {
				Log     bool   `json:"log"`
				Level   int    `json:"level"` // 1-3
				Channel string `json:"channel"`
			} `json:"logger"`
		} `json:"modules"`
	}
)

func setupLogging(discord *dgofw.DiscordClient, level int, ch string) {
	if level >= 1 {
		discord.WithGuildBanAdd(false, func(ban *dgofw.DiscordGuildBan) {
			discord.Send(ch, ban.User.Username()+" was banned!")
		})

		discord.WithGuildBanRemove(false, func(ban *dgofw.DiscordGuildBan) {
			discord.Send(ch, ban.User.Username()+" was unbanned!")
		})

		discord.OnMemberAdd(false, func(mem *dgofw.DiscordMember) {
			discord.Send(ch, mem.User.Username()+" just joined the server!")
		})

		discord.OnMemberRemove(false, func(mem *dgofw.DiscordMember) {
			discord.Send(ch, mem.User.Username()+" left the server!")
		})
	}
}

func (c *Config) buildCommand(name string, args ...string) string {
	var buf bytes.Buffer
	buf.WriteString(c.Modules.Discord.Prefix + name)
	if len(args) == 0 {
		return buf.String()
	}
	buf.WriteString(" ")
	for i := range args {
		args[i] = "{" + args[i] + "}"
	}
	buf.WriteString(strings.Join(args, " "))
	return buf.String()
}

func main() {
	var cfg Config
	b, err := ioutil.ReadFile("./config.json")
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		panic(err)
	}

	discord := dgofw.NewDiscordClient(cfg.Modules.Discord.Token)

	if cfg.Modules.Logging.Log && cfg.Modules.Logging.Channel != "" {
		setupLogging(discord, cfg.Modules.Logging.Level, cfg.Modules.Logging.Channel)
	}

	gfyc := gfycat.NewGfyCatPlugin(cfg.Modules.Gfycat.ClientID, cfg.Modules.Gfycat.ClientSecret, gofycat.Client)
	lfmc := lastfm.NewLastFMClient(cfg.Modules.LastFM.AppID)
	opeth := opeth.NewOpethPlugin()
	wap := wolframplugin.NewWolframPlugin(cfg.Modules.Wolfram.AppID)
	quotes := quoteplugin.NewQuotePlugin()
	tagsp := tags.NewTags()
	sptfy := spotifyplugin.NewSpotifyPlugin(cfg.Modules.Spotify.ClientID, cfg.Modules.Spotify.ClientSecret)
	musicc := music.NewMusicPlayer(discord)
	reddit := memes.NewMemer(runtime.GOOS + ":mountainbot:v0.1: (by /u/Krognol)")
	weebc := malist.NewWeebClient(
		cfg.Modules.Weebery.Anilist.ClientID,
		cfg.Modules.Weebery.Anilist.ClientSecret,
		cfg.Modules.Weebery.MAL.Username,
		cfg.Modules.Weebery.MAL.Password,
	)

	discord.OnReady(true, func(r *discordgo.Ready) {
		discord.SetStatus(cfg.Modules.Discord.Prefix + "help")
	})

	discord.OnMessage(cfg.buildCommand("roll", "num"), false, func(m *dgofw.DiscordMessage) {
		if i, err := strconv.ParseInt(m.Arg("num"), 0, 64); err == nil {
			m.Reply(fmt.Sprintf("%d", rand.Int63n(i+1)))
		} else {
			m.Reply(fmt.Sprintf("%d", rand.Int()))
		}
	})
	discord.OnMessage("meme", false, reddit.OnDankMeme)
	// This is just a markov chain, just edit the plugin to work for other files
	discord.OnMessage(cfg.buildCommand("opeth"), false, opeth.OnMessage)
	discord.OnMessage(cfg.buildCommand("urban", "word"), false, udplugin.UrbanOnMessage)
	discord.OnMessage(cfg.buildCommand("define", "word"), false, wiktionaryplugin.WikiOnMessage)
	discord.OnMessage(cfg.buildCommand("wholesomememe"), false, reddit.OnWholesomeMeme)
	discord.OnMessage(cfg.buildCommand("wolfram", "query"), false, wap.OnMessage)
	discord.OnMessage(cfg.buildCommand("fm", "arg1", "arg2"), false, lfmc.OnMessage)
	discord.OnMessage(cfg.buildCommand("gfy", "arg1", "arg2"), false, gfyc.OnMessage)
	discord.OnMessage(cfg.buildCommand("mal", "arg1", "arg2"), false, weebc.MALOnMessage)
	discord.OnMessage(cfg.buildCommand("info", "arg1", "arg2"), false, userinfo.UIOnMessage)
	discord.OnMessage(cfg.buildCommand("quote", "arg1", "arg2"), false, quotes.OnMessage)
	discord.OnMessage(cfg.buildCommand("music", "arg1", "arg2"), false, musicc.OnMessage)
	discord.OnMessage(cfg.buildCommand("anilist", "arg1", "arg2"), false, weebc.AnilistOnMessage)
	discord.OnMessage(cfg.buildCommand("ow", "battletag", "region"), false, owplugin.OWOnMessage)
	discord.OnMessage(cfg.buildCommand("tag", "arg1", "arg2", "arg3"), false, tagsp.OnMessage)
	discord.OnMessage(cfg.buildCommand("spotify", "arg1", "arg2", "arg3"), false, sptfy.OnMessage)

	discord.OnMessage(cfg.buildCommand("ping"), false, func(m *dgofw.DiscordMessage) {
		m.Reply("pong!")
	})

	discord.OnMessage(cfg.buildCommand("lenny"), false, func(m *dgofw.DiscordMessage) {
		m.Reply(lenny.GetLenny())
	})

	discord.OnMessage(cfg.buildCommand("choose", "options"), false, func(m *dgofw.DiscordMessage) {
		options := strings.Split(m.Arg("options"), "|")
		if len(options) == 0 {
			return
		}

		if len(options) < 2 {
			m.Reply(options[0])
			return
		}

		rand.Seed(time.Now().UnixNano())
		i := rand.Intn(len(options) - 1)
		m.Reply(options[i])
	})

	discord.OnMessage(cfg.buildCommand("cowsay", "text"), false, func(m *dgofw.DiscordMessage) {
		if text := m.Arg("text"); text != "" {
			res, err := http.Get("http://cowsay.morecode.org/say?format=json&message=" + url.QueryEscape(text))
			if err == nil {
				type temp struct {
					Cow string `json:"cow"`
				}
				var cow temp
				defer res.Body.Close()
				err = json.NewDecoder(res.Body).Decode(&cow)
				if err != nil {
					return
				}
				m.Reply("```\n" + cow.Cow + "\n```")
			}
		}
	})

	discord.OnMessage(cfg.buildCommand("help", "mod"), false, func(m *dgofw.DiscordMessage) {
		mod := m.Arg("mod")
		if mod == "" {
			m.Reply("use `" + cfg.Modules.Discord.Prefix + "help [thing]`\nThings:'gfy', 'fm', 'mal', 'wiki', 'urban',\n'wolfram', 'tags', 'quotes', 'ow', 'userinfo', 'spotify', 'other'")
			return
		}
		var help string
		switch mod {
		case "gfy", "gif", "gfycat":
			help = strings.Join(gfycat.GfyHelp, "\n")
		case "fm", "lf", "last fm":
			help = strings.Join(lastfm.LfmHelp, "\n")
		case "mal", "anilist", "anime", "manga", "mango", "weeb":
			help = strings.Join(malist.MALHelp, "\n")
		case "wiki", "define", "wiktionary":
			help = "define [word] -- Gets a word definition from wiktionary."
		case "ud", "urban", "xd":
			help = "urban [word] -- Urban dictionary definition of a word"
		case "wolfram":
			help = "wolfram [query] -- Queries wolfram|alpha"
		case "tags", "tag":
			help = strings.Join(tags.TagsHelp, "\n")
		case "quotes", "quote":
			help = strings.Join(quoteplugin.QuotesHelp, "\n")
		case "ow", "overwatch":
			help = "ow [battletag] [region] -- OW stats"
		case "userinfo":
			help = strings.Join(userinfo.UserinfoHelp, "\n")
		case "spotify":
			help = strings.Join(spotifyplugin.SpotifyHelp, "\n")
		case "music":
			help = strings.Join(music.MusicHelp, "\n")
		case "other":
			help = "lenny -- Random lenny face\nping -- Pong!\ncowsay [text] -- Moo\nroll [N] -- Rolls a random number between 0..N\nmeme -- dank meme\nwholesomememe -- FeelsOkMan"
		}
		m.Reply(help)
	})

	discord.Connect()

	close := make(chan os.Signal, 1)
	signal.Notify(close, os.Interrupt, os.Kill)

	<-close
	discord.Disconnect()
	os.Exit(0)
}
