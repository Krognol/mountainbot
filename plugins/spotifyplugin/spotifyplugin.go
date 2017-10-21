package spotifyplugin

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/Krognol/dgofw"
	"github.com/Krognol/go-spotify/spotify"
)

type SpotifyClient struct {
	Client *spotify.Client
}

func NewSpotifyPlugin(clientid, clientsecret string) *SpotifyClient {
	return &SpotifyClient{
		Client: spotify.New(clientid, clientsecret),
	}
}

var SpotifyHelp = []string{
	"spotify search track [query] -- Searches spotify for a track",
	"spotify search album [query] -- Searches spotify for an album",
	"spotify search artist [query] -- Searches spotify for a band or artist",
}

func (s *SpotifyClient) waitForT(m *dgofw.DiscordMessage, t interface{}) {
	var buf bytes.Buffer
	var tlen int
	buf.WriteString("I have more than 5 results. \nPlease pick one.\n\n")
	switch typ := t.(type) {
	case []*spotify.Track:
		tlen = len(typ)
		for i, track := range typ {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s** by **%s**\n", i+1, track.Name, track.Artists[0].Name))
			if i >= 4 {
				break
			}
		}
	case []*spotify.Artist:
		tlen = len(typ)
		for i, artist := range typ {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s**\n", i+1, artist.Name))
			if i >= 4 {
				break
			}
		}
	case []*spotify.Album:
		tlen = len(typ)
		for i, album := range typ {
			buf.WriteString(fmt.Sprintf("`%d`\t**%s** by **%s**\n", i+1, album.Name, album.Artists[0].Name))
			if i >= 4 {
				break
			}
		}
	default:
		return
	}

	m2 := m.Reply(buf.String())
	m2.WaitForMessage(15, func(itr *dgofw.DiscordMessage) bool {
		if itr.Author.ID() == m.Author.ID() && itr.ChannelID() == m.ChannelID() {
			if i, err := strconv.ParseInt(itr.Content(), 0, 64); err == nil {
				if i > 0 && int(i) < tlen {
					switch elements := t.(type) {
					case []*spotify.Track:
						m2.Edit("https://open.spotify.com/track/" + elements[i-1].ID)
					case []*spotify.Album:
						m2.Edit("https://open.spotify.com/album/" + elements[i-1].ID)
					case []*spotify.Artist:
						m2.Edit("https://open.spotify.com/artist/" + elements[i-1].ID)
					}
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

func (s *SpotifyClient) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	switch arg1 {
	case "search":
		if query := m.Arg("arg3"); query != "" {
			typ := m.Arg("arg2")
			switch typ {
			case "track":
				tracks, err := s.Client.SearchTrack(query, 0)
				if err != nil {
					fmt.Println(err)
					m.Reply("Something happened...")
					return
				}

				if len(tracks) == 0 {
					m.Reply("No results")
					return
				}

				if len(tracks) < 5 {
					m.Reply("https://open.spotify.com/track/" + tracks[0].ID)
					return
				}

				s.waitForT(m, tracks)
			case "artist":
				artists, err := s.Client.SearchArtist(query, 0)
				if err != nil {
					fmt.Println(err)
					m.Reply("Something happened...")
					return
				}

				if len(artists) == 0 {
					m.Reply("No results")
					return
				}

				if len(artists) < 5 {
					m.Reply("https://open.spotify.com/artist/" + artists[0].ID)
					return
				}

				s.waitForT(m, artists)
			case "album":
				albums, err := s.Client.SearchAlbum(query, 0)
				if err != nil {
					fmt.Println(err)
					m.Reply("Something happened...")
					return
				}

				if len(albums) == 0 {
					m.Reply("No results")
					return
				}

				if len(albums) < 5 {
					m.Reply("https://open.spotify.com/album/" + albums[0].ID)
					return
				}

				s.waitForT(m, albums)
			}
		}
	}
}
