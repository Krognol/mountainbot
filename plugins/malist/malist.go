package malist

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"nano/plugins/mal"
	"net/http"
	"strings"

	"fmt"
	"net/url"

	"strconv"

	"github.com/Krognol/dgofw"
	"github.com/bwmarrin/discordgo"
)

type WeebClient struct {
	AnilistClientID     string
	AnilistClientSecret string

	// For basic Auth
	MALClient *mal.MALClient
}

var MALHelp = []string{
	"(mal | anilist) anime [name] -- lookup an anime on MAL. If it doesn't find it on there it tries Anilist instead.",
	"(mal | anilist) manga [name] -- lookup manga on MAL. If it doesn't find it on there it tries Anilist instead.",
}

const anilistURL = "https://anilist.co/api/"

func NewWeebClient(anilistcid, anilistsec, maluname, malpw string) *WeebClient {
	return &WeebClient{
		AnilistClientID:     anilistcid,
		AnilistClientSecret: anilistsec,
		MALClient: &mal.MALClient{
			Username: maluname,
			Password: malpw,
		},
	}
}

func (c *WeebClient) anilistAuth() (string, error) {
	url := fmt.Sprintf("%sauth/access_token?grant_type=client_credentials&client_id=%s&client_secret=%s", anilistURL, c.AnilistClientID, c.AnilistClientSecret)
	req, _ := http.NewRequest("POST", url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	type Temp struct {
		Token string `json:"access_token"`
	}

	var t Temp
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&t)
	if err != nil {
		return "", err
	}

	return t.Token, nil
}

func (c *WeebClient) anilistSearchRequest(method, name string) (*http.Response, error) {
	if strings.Contains(name, "/") {
		name = strings.Replace(name, "/", " ", -1)
	}
	name = url.QueryEscape(name)
	token, err := c.anilistAuth()
	if err != nil {
		fmt.Println("Couldn't get token", err)
		return nil, err
	} else if token == "" {
		return nil, errors.New("Couldn't get token")
	}

	return http.Get(anilistURL + method + name + "?access_token=" + token)
}

type AnilistResult struct {
	ID               int      `json:"id"`
	TitleRomaji      string   `json:"title_romaji"`
	TitleJapanese    string   `json:"title_japanese"`
	ImageURLLarge    string   `json:"image_url_lge"`
	SeriesType       string   `json:"series_type"`
	AiringStatus     string   `json:"airing_status"`
	PublishingStatus string   `json:"publishing_status"`
	TotalEpisodes    int      `json:"total_episodes"`
	TotalVolumes     int      `json:"total_volumes"`
	TotalChapters    int      `json:"total_chapters"`
	AverageScore     int      `json:"average_score"`
	Genres           []string `json:"genres"`
	Synonyms         []string `json:"synonyms"`
	Source           string   `json:"source"`
	Description      string   `json:"description"`
}

func unmarshal(res *http.Response, v interface{}) error {
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(v)
}

func newAnilistAnimeEmbed(anime *AnilistResult) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Description: strings.Replace(anime.Description, "<br>", "\n", -1),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Anilist.co"},
		Author: &discordgo.MessageEmbedAuthor{
			URL:     fmt.Sprintf("https://anilist.co/anime/%d/%s", anime.ID, url.QueryEscape(anime.TitleRomaji)),
			IconURL: "https://anilist.co/favicon.ico",
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: anime.ImageURLLarge,
		},
		Fields: append(make([]*discordgo.MessageEmbedField, 0),
			&discordgo.MessageEmbedField{
				Name:   "Type",
				Value:  anime.SeriesType,
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Status",
				Value:  anime.AiringStatus,
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Episodes",
				Value:  fmt.Sprintf("%d", anime.TotalEpisodes),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Average Score",
				Value:  fmt.Sprintf("%d/100", anime.AverageScore),
				Inline: true,
			},
		),
	}

	if anime.TitleJapanese != "" {
		embed.Author.Name = anime.TitleJapanese
	} else {
		embed.Author.Name = anime.TitleRomaji
	}

	if anime.Source != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Source",
			Value:  anime.Source,
			Inline: true,
		})
	}

	gensy := []string{}
	for _, genre := range anime.Genres {
		if genre != "" {
			gensy = append(gensy, genre)
		}
	}
	if len(gensy) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Genres",
			Value:  strings.Join(gensy, ", "),
			Inline: true,
		})
	}
	gensy = []string{}
	for _, synonym := range anime.Synonyms {
		if synonym != "" {
			gensy = append(gensy, synonym)
		}
	}
	if len(gensy) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Synonyms",
			Value:  strings.Join(gensy, ",\n"),
			Inline: true,
		})
	}
	return embed
}

func newAnilistMangaEmbed(manga *AnilistResult) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: "https://anilist.co/favicon.ico",
			URL:     fmt.Sprintf("https://anilist.co/manga/%d/%s", manga.ID, url.QueryEscape(manga.TitleRomaji)),
		},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Anilist.co"},
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: manga.ImageURLLarge},
		Description: strings.Replace(manga.Description, "<br>", "\n", -1),
		Fields: append(make([]*discordgo.MessageEmbedField, 0),
			&discordgo.MessageEmbedField{
				Name:   "Type",
				Value:  manga.SeriesType,
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Status",
				Value:  manga.PublishingStatus,
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Volumes",
				Value:  fmt.Sprintf("%d", manga.TotalVolumes),
				Inline: true,
			}, &discordgo.MessageEmbedField{
				Name:   "Chapters",
				Value:  fmt.Sprintf("%d", manga.TotalChapters),
				Inline: true,
			},
		),
	}

	if manga.TitleJapanese != "" {
		embed.Author.Name = manga.TitleJapanese
	} else {
		embed.Author.Name = manga.TitleRomaji
	}

	gensy := []string{}
	for _, genre := range manga.Genres {
		if genre != "" {
			gensy = append(gensy, genre)
		}
	}
	if len(gensy) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Genres",
			Value:  strings.Join(gensy, ", "),
			Inline: true,
		})
	}

	gensy = []string{}

	for _, syn := range manga.Synonyms {
		if syn != "" {
			gensy = append(gensy, syn)
		}
	}
	if len(gensy) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Synonyms",
			Value:  strings.Join(gensy, ",\n"),
			Inline: true,
		})
	}
	return embed
}

func (c *WeebClient) getAnilistAnime(m *dgofw.DiscordMessage, name string) {
	res, err := c.anilistSearchRequest("anime/search/", name)
	if err != nil {
		m.Reply("Encountered some error...")
		fmt.Println(err)
		return
	}

	var animes []*AnilistResult

	if err := unmarshal(res, &animes); err != nil {
		m.Reply("Something happened...")
		fmt.Println(err)
		return
	}

	if len(animes) == 0 {
		m.Reply("No results")
		return
	}

	if len(animes) > 9 {
		c.waitForT(m, animes)
		return
	}
	embed := newAnilistAnimeEmbed(animes[0])
	embed.Color = m.Session().State.UserColor(m.Author.ID(), m.ChannelID())
	m.ReplyEmbed(embed)
}

func (c *WeebClient) getAnilistManga(m *dgofw.DiscordMessage, name string) {
	res, err := c.anilistSearchRequest("manga/search/", name)
	if err != nil {
		m.Reply("Encountered some error...")
		fmt.Println(err)
		return
	}

	var mangas []*AnilistResult
	if err := unmarshal(res, &mangas); err != nil {
		m.Reply("Something happened...")
		fmt.Println(err)
		return
	}

	if len(mangas) == 0 {
		m.Reply("No results")
		return
	}

	if len(mangas) > 9 {
		c.waitForT(m, mangas)
		return
	}

	mango := newAnilistMangaEmbed(mangas[0])
	mango.Color = m.Session().State.UserColor(m.Author.ID(), m.ChannelID())

	m.ReplyEmbed(mango)
}

func newMalMangaEmbed(first mal.Entry) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: "https://myanimelist.cdn-dena.com/images/faviconv5.ico",
			Name:    first.Title,
			URL:     fmt.Sprintf("https://myanimelist.net/anime/%d/%s", first.Id, strings.Replace(first.Title, " ", "_", -1)),
		},
		Description: strings.Replace(html.UnescapeString(first.Synopsis), "<br />", "", -1),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%s | Status: %s | Episodes: %d | Score: %s", first.Type, first.Status, first.Episodes, strconv.FormatFloat(first.Score, 'f', 2, 64)),
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: first.Image,
		},
	}
}

func newMalAnimeEmbed(first mal.Entry) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: "https://myanimelist.cdn-dena.com/images/faviconv5.ico",
			Name:    first.Title,
			URL:     fmt.Sprintf("https://myanimelist.net/anime/%d/%s", first.Id, strings.Replace(first.Title, " ", "_", -1)),
		},
		Description: strings.Replace(html.UnescapeString(first.Synopsis), "<br />", "", -1),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%s | Status: %s | Episodes: %d | Score: %s", first.Type, first.Status, first.Episodes, strconv.FormatFloat(first.Score, 'f', 2, 64)),
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: first.Image,
		},
	}
}

func (c *WeebClient) getMALAnime(m *dgofw.DiscordMessage, name string) {
	res := c.MALClient.GetAnime(name)
	if res == nil || len(res.Entries) == 0 {
		c.getAnilistAnime(m, name)
		return
	}

	if len(res.Entries) > 9 {
		c.waitForT(m, &res)
		return
	}

	animu := newMalAnimeEmbed(res.Entries[0])
	animu.Color = m.Session().State.UserColor(m.Author.ID(), m.ChannelID())
	m.ReplyEmbed(animu)

}

func (c *WeebClient) getMALManga(m *dgofw.DiscordMessage, name string) {
	res := c.MALClient.GetManga(name)
	if res == nil || len(res.Entries) == 0 {
		c.getAnilistManga(m, name)
		return
	}

	if len(res.Entries) > 9 {
		c.waitForT(m, &res)
		return
	}

	mango := newMalMangaEmbed(res.Entries[0])
	mango.Color = m.Session().State.UserColor(m.Author.ID(), m.ChannelID())
	m.ReplyEmbed(mango)
}

func (c *WeebClient) waitForT(m *dgofw.DiscordMessage, t interface{}) {
	var embed *discordgo.MessageEmbed
	var buf bytes.Buffer
	var elemLen int
	buf.WriteString("I have more than 10 results.\nPlease pick one.\n\n")
	switch typ := t.(type) {
	case *mal.Anime:
		elemLen = len(typ.Entries)
		for i, entry := range typ.Entries {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s**\n", i+1, entry.Title))
			if i >= 9 {
				break
			}
		}
	case *mal.Manga:
		elemLen = len(typ.Entries)
		for i, entry := range typ.Entries {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s**\n", i+1, entry.Title))
			if i >= 9 {
				break
			}
		}
	case []*AnilistResult:
		elemLen = len(typ)
		for i, entry := range typ {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s**\n", i+1, entry.TitleRomaji))
			if i >= 9 {
				break
			}
		}
	}

	m2 := m.Reply(buf.String())
	m2.WaitForMessage(15, func(itr *dgofw.DiscordMessage) bool {
		if itr.Author.ID() == m.Author.ID() && itr.ChannelID() == m.ChannelID() {
			if i, err := strconv.ParseInt(itr.Content(), 0, 64); err == nil {
				if i > 0 && int(i) < elemLen {
					switch typ := t.(type) {
					case *mal.Anime:
						embed = newMalAnimeEmbed(typ.Entries[i-1])
					case *mal.Manga:
						embed = newMalMangaEmbed(typ.Entries[i-1])
					case []*AnilistResult:
						embed = newAnilistAnimeEmbed(typ[i-1])
					}
					embed.Color = m.Session().State.UserColor(m.Author.ID(), m.ChannelID())
					m2.EditEmbed(embed)
					itr.Delete()
					return true
				}
			}
		}
		return false
	}, func() {
		m.DeleteMany(m, m2)
	})
}

func (c *WeebClient) MALOnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	name := m.Arg("arg2")
	switch arg1 {
	case "anime":
		c.getMALAnime(m, name)
	case "manga":
		c.getMALManga(m, name)
	}
}

func (c *WeebClient) AnilistOnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	name := m.Arg("arg2")
	switch arg1 {
	case "anime":
		c.getAnilistAnime(m, name)
	case "manga":
		c.getAnilistManga(m, name)
	}
}
