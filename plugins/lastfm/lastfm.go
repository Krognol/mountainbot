package lastfm

import (
	"bytes"
	"net/http"
	"sync"

	"fmt"

	"encoding/json"

	"os"

	"io/ioutil"

	"github.com/Krognol/dgofw"
	"github.com/bwmarrin/discordgo"
)

type LastfmPlugin struct {
	sync.RWMutex
	APIKey      string            `json:"api_key"`
	CachedUsers map[string]string `json:"cached_users"`
}

const collageURL = "http://lastfmtopalbums.dinduks.com/patchwork.php?period=7day&rows=4&cols=4&imageSize=250&user=%s"

type Recent struct {
	RecentTracks struct {
		Track []struct {
			Artist struct {
				Text string `json:"#text"`
				MbID string `json:"mbid"`
			} `json:"artist"`
			Name       string `json:"name"`
			Streamable string `json:"streamable"`
			MbID       string `json:"mbid"`
			Album      struct {
				Text string `json:"#text"`
				MbID string `json:"mbid"`
			} `json:"album"`
			URL   string `json:"url"`
			Image []struct {
				Text string `json:"#text"`
				Size string `json:"size"`
			} `json:"image"`
			Attr struct {
				NowPlaying string `json:"nowplaying"`
			} `json:"@attr,omitempty"`
			Date struct {
				Uts  string `json:"uts"`
				Text string `json:"#text"`
			} `json:"date,omitempty"`
		} `json:"track"`
		Attr struct {
			User       string `json:"user"`
			Page       string `json:"page"`
			PerPage    string `json:"perPage"`
			TotalPages string `json:"totalPages"`
			Total      string `json:"total"`
		} `json:"@attr"`
	} `json:"recenttracks"`
}

func NewLastFMClient(apikey string) *LastfmPlugin {
	plugin := &LastfmPlugin{
		APIKey:      apikey,
		CachedUsers: make(map[string]string),
	}
	err := plugin.Load()
	if err != nil {
		fmt.Println(err.Error())
		plugin.APIKey = apikey
		plugin.CachedUsers = make(map[string]string)
		plugin.Save()
	}
	return plugin
}

func (l *LastfmPlugin) request(method, user, span string, limit int) (*http.Response, error) {
	url := fmt.Sprintf("https://ws.audioscrobbler.com/2.0/?method=%s&format=json&user=%s&api_key=%s",
		method,
		user,
		l.APIKey,
	)
	if span != "" {
		url += "&period=" + span
	}
	if limit != 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}
	req, _ := http.NewRequest("GET", url, nil)
	return http.DefaultClient.Do(req)
}

var LfmHelp = []string{
	"fm set [name]    -- Sets a last.fm username in the local cache",
	"fm now                -- Looks up last.fm users currently playing track. \n\t\t\t\t               If no username is given takes from local cache",
	"fm recent            -- Gets the users last 5 played songs.",
	"fm collage           -- Gets a 4x4 collage of the users top played albums the last 7 days",

	"fm top weekly    -- Weekly top tracks for the user",
	"fm top tracks      -- All time top tracks for the user",
	"fm top albums    -- All time top albums for the user",
	"fm top artists      -- All time top artists for the user",
}

func unmarshal(b *http.Response, v interface{}) error {
	defer b.Body.Close()
	return json.NewDecoder(b.Body).Decode(v)
}

func (l *LastfmPlugin) fmNow(m *dgofw.DiscordMessage, u string) {
	res, err := l.request("user.getrecenttracks", u, "", 1)
	if err != nil {
		m.Reply("Something happened...")
		return
	}
	var recent Recent
	err = unmarshal(res, &recent)
	if err != nil || len(recent.RecentTracks.Track) == 0 {
		m.Reply("Something happened...")
		return
	}
	track := recent.RecentTracks.Track[0]
	if track.Attr.NowPlaying == "" {
		m.Reply("Looks like you're not scrobbling a track right now")
		return
	}
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    recent.RecentTracks.Attr.User,
			URL:     "https://last.fm/user/" + recent.RecentTracks.Attr.User,
			IconURL: m.Author.Avatar(),
		},
		Title: fmt.Sprintf("%s -- %s", track.Name, track.Artist.Text),
		URL:   track.URL,
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
	}
	if track.Image[3].Text != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: track.Image[3].Text,
		}
	}

	m.ReplyEmbed(embed)
}

func (l *LastfmPlugin) fmRecent(m *dgofw.DiscordMessage, u string) {
	res, err := l.request("user.getrecenttracks", u, "", 11)
	if err != nil {
		return
	}

	var recent Recent
	if err = unmarshal(res, &recent); err != nil {
		m.Reply("Something happened...")
		return
	}
	tracks := recent.RecentTracks.Track

	embed := &discordgo.MessageEmbed{
		Color:       m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
		Description: "",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://last.fm/user/" + u,
			IconURL: m.Author.Avatar(),
			Name:    u,
		},
	}
	var imgi int
	if tracks[0].Attr.NowPlaying != "" {
		imgi = 1
	} else {
		imgi = 0
	}
	var buf bytes.Buffer
	for i, track := range tracks[imgi:] {
		artist := track.Artist.Text
		song := track.Name
		url := track.URL
		if i == 9 {
			buf.WriteString(fmt.Sprintf("`%d`\t  **[%s](%s)** by **%s**\n", (i + 1), song, url, artist))
			break
		}
		buf.WriteString(fmt.Sprintf("`%d`\t    **[%s](%s)** by **%s**\n", (i + 1), song, url, artist))
	}
	embed.Description = buf.String()
	if tracks[imgi].Image[3].Text == "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "http://i.imgur.com/jzZ5llc.png",
		}
	} else {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: tracks[imgi].Image[3].Text,
		}
	}
	m.ReplyEmbed(embed)
}

type Top struct {
	TopTracks struct {
		Track []struct {
			Artist struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"artist"`
			Name      string `json:"name"`
			URL       string `json:"url"`
			Playcount string `json:"playcount"`
			Image     []struct {
				URL string `json:"#text"`
			} `json:"image"`
		} `json:"track"`
	} `json:"toptracks"`
}

func (l *LastfmPlugin) fmTopTracks(m *dgofw.DiscordMessage, u, span string) {
	res, err := l.request("user.gettoptracks", u, span, 10)
	if err != nil {
		fmt.Println(err)
		return
	}

	var tops Top
	err = unmarshal(res, &tops)
	if err != nil {
		fmt.Println(err)
		return
	}

	embed := &discordgo.MessageEmbed{
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    u + "'s Weekly Top 10",
			IconURL: m.Author.Avatar(),
			URL:     "https://last.fm/user/" + u,
		},
		Description: "",
	}

	var buf bytes.Buffer
	for i, track := range tops.TopTracks.Track {
		artist := track.Artist.Name

		song := track.Name
		songURL := track.URL

		if i == 9 {
			buf.WriteString(fmt.Sprintf("`%d`\t  **[%s](%s)** by **%s** (%s plays)\n", (i + 1), song, songURL, artist, track.Playcount))
		} else {
			buf.WriteString(fmt.Sprintf("`%d`\t    **[%s](%s)** by **%s** (%s plays)\n", (i + 1), song, songURL, artist, track.Playcount))
		}
	}
	embed.Description = buf.String()

	if tops.TopTracks.Track[0].Image[3].URL == "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "http://i.imgur.com/jzZ5llc.png",
		}
	} else {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: tops.TopTracks.Track[0].Image[3].URL,
		}
	}
	m.ReplyEmbed(embed)
}

type TopAlbum struct {
	TopAlbums struct {
		Album []struct {
			Name      string `json:"name"`
			Playcount string `json:"playcount"`
			URL       string `json:"url"`
			Artist    struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"url"`
			Image []struct {
				URL string `json:"#text"`
			}
		} `json:"album"`
	} `json:"topalbums"`
}

func (l *LastfmPlugin) fmTopAlbums(m *dgofw.DiscordMessage, u string) {
	res, err := l.request("user.gettopalbums", u, "", 10)
	if err != nil {
		return
	}

	var tops TopAlbum
	err = unmarshal(res, &tops)
	if err != nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    u + "'s Top Albums",
			IconURL: m.Author.Avatar(),
			URL:     "https://last.fm/user/" + u,
		},
		Description: "",
	}

	var buf bytes.Buffer

	for i, album := range tops.TopAlbums.Album {
		name := album.Name
		plays := album.Playcount
		url := album.URL
		artist := album.Artist.Name
		artistURL := album.Artist.URL

		if i == 9 {
			buf.WriteString(fmt.Sprintf("`%d`\t  **[%s](%s)** by **[%s](%s)** (%s plays)\n", (i + 1), name, url, artist, artistURL, plays))
		} else {
			buf.WriteString(fmt.Sprintf("`%d`\t    **[%s](%s)** by **[%s](%s)** (%s plays)\n", (i + 1), name, url, artist, artistURL, plays))
		}
	}
	embed.Description = buf.String()

	if tops.TopAlbums.Album[0].Image[3].URL == "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "http://i.imgur.com/jzZ5llc.png",
		}
	} else {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: tops.TopAlbums.Album[0].Image[3].URL,
		}
	}
	m.ReplyEmbed(embed)
}

type TopArtist struct {
	TopArtists struct {
		Artist []struct {
			Name      string `json:"name"`
			Playcount string `json:"playcount"`
			URL       string `json:"url"`
			Image     []struct {
				URL string `json:"#text"`
			} `json:"image"`
		} `json:"artist"`
	} `json:"topartists"`
}

func (l *LastfmPlugin) fmTopArtists(m *dgofw.DiscordMessage, u string) {
	res, err := l.request("user.gettopartists", u, "", 10)
	if err != nil {
		return
	}

	var tops TopArtist
	err = unmarshal(res, &tops)
	if err != nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Color: m.Session().State.UserColor(m.Author.ID(), m.ChannelID()),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    u + "'s Top Artists",
			URL:     "https://last.fm/user" + u,
			IconURL: m.Author.Avatar(),
		},
		Description: "",
	}

	var buf bytes.Buffer

	for i, artist := range tops.TopArtists.Artist {
		if i == 9 {
			buf.WriteString(fmt.Sprintf("`%d`\t  **[%s](%s)**(%s plays)\n", (i + 1), artist.Name, artist.URL, artist.Playcount))
		} else {
			buf.WriteString(fmt.Sprintf("`%d`\t    **[%s](%s)**(%s plays)\n", (i + 1), artist.Name, artist.URL, artist.Playcount))
		}
	}

	embed.Description = buf.String()

	if tops.TopArtists.Artist[0].Image[3].URL == "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "http://i.imgur.com/jzZ5llc.png",
		}
	} else {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: tops.TopArtists.Artist[0].Image[3].URL,
		}
	}

	m.ReplyEmbed(embed)
}

func (l *LastfmPlugin) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	if arg1 == "set" {
		if arg2 := m.Arg("arg2"); arg2 != "" {
			l.CachedUsers[m.Author.ID()] = arg2
			l.Save()
			m.Reply("Set Last.FM username for '" + m.Author.Mention() + "' to " + arg2)
		}
		return
	}
	u, ok := l.CachedUsers[m.Author.ID()]
	if !ok {
		m.Reply("You don't have a Last.FM username set!\nYou can set it with `fm set [username]`")
		return
	}
	switch arg1 {
	case "now":
		l.fmNow(m, u)
	case "recent":
		l.fmRecent(m, u)
	case "collage":
		res, err := http.Get(fmt.Sprintf(collageURL, u))
		if err != nil {
			m.Reply("Couldn't get collage :(")
			return
		}
		m.ReplyFileWithMessage(m.Author.Mention(), u+"_lfm_collage_250x250_4x4.png", res.Body)
	case "top":
		arg2 := m.Arg("arg2")
		switch arg2 {
		case "weekly":
			l.fmTopTracks(m, u, "7day")
		case "tracks":
			l.fmTopTracks(m, u, "")
		case "albums":
			l.fmTopAlbums(m, u)
		case "artists":
			l.fmTopArtists(m, u)
		}
	}
}

func (p *LastfmPlugin) Save() (err error) {
	var f *os.File
	if f, err = os.Create("./lastfmstate.json"); err == nil {
		defer f.Close()
		return json.NewEncoder(f).Encode(p)
	}
	return
}

func (p *LastfmPlugin) Load() (err error) {
	var b []byte
	if b, err = ioutil.ReadFile("./lastfmstate.json"); err == nil {
		return json.Unmarshal(b, p)
	}
	return
}
