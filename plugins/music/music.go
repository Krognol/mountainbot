package music

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/Krognol/dgofw"
)

type (
	controlMessage int
	Tracks         []*Track
	Track          struct {
		AddedBy     *dgofw.DiscordUser
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		FullTitle   string `json:"full_title"`
		Thumbnail   string `json:"thumbnail"`
		URL         string `json:"url"`
		Duration    int    `json:"duration"`
		LenMinutes  int
		LenSeconds  int
		Remaining   int
	}

	Connection struct {
		sync.Mutex

		Guild        string
		Channel      string
		MaxQueueSize int
		Queue        Tracks

		close   chan struct{}
		control chan controlMessage
		status  controlMessage

		current *Track
		conn    *dgofw.DiscordVoiceConnection
	}

	MusicPlayer struct {
		sync.Mutex
		discord          *dgofw.DiscordClient
		VoiceConnections map[string]*Connection
	}
)

func (t Tracks) Len() int {
	return len(t)
}

func (t Tracks) Less(i, j int) bool {
	rand.Seed(time.Now().UnixNano() + int64((i + 2*2)))
	return rand.Intn(2) == 1
}

func (t Tracks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

const (
	Skip controlMessage = iota
	Pause
	Play
	Resume
	Stop
)

var MusicHelp = []string{
	"music join -- Joins a voice channel",
	"music leave -- Leaves a voice channel",

	"music skip -- Skips a track",
	"music pause -- Pauses a track",
	"music resume -- Unpauses a track",
	"music shuffle -- Shuffles the queue.",
	"music play [song] -- Plays a track. Has to be in a voice channel. Queues it if a track is already playing.",
}

func NewMusicPlayer(client *dgofw.DiscordClient) *MusicPlayer {
	return &MusicPlayer{
		discord:          client,
		VoiceConnections: make(map[string]*Connection),
	}
}

func (mp *MusicPlayer) play(vc *Connection, track *Track) {
	var ytdl *exec.Cmd

	// Had to do this because of where it was hosted
	if runtime.GOOS == "windows" {
		ytdl = exec.Command("./youtube-dl", "-v", "-f", "bestaudio", "-o", "-", track.URL)
	} else if runtime.GOOS == "linux" {
		ytdl = exec.Command("python3", "youtube-dl", "-v", "-f", "bestaudio", "-o", "-", track.URL)
	}

	ytdlout, err := ytdl.StdoutPipe()
	if err != nil {
		fmt.Println("Failed ytdlout\n", err)
		return
	}

	ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)

	dca := exec.Command("./dca", "-raw", "-vol", "128", "-i", "pipe:0")
	dca.Stdin = ytdlbuf

	dcaout, err := dca.StdoutPipe()
	if err != nil {
		fmt.Println("Failed dcaout\n", err)
		return
	}

	dcabuf := bufio.NewReaderSize(dcaout, 16384)

	err = ytdl.Start()
	if err != nil {
		fmt.Println("Failed ytdl.Start\n", err)
		return
	}

	defer func() {
		go ytdl.Wait()
	}()

	err = dca.Start()
	if err != nil {
		fmt.Println("Failed dca.Start\n", err)
		return
	}

	defer func() {
		go dca.Wait()
	}()

	var opusLen int16

	vc.conn.Speaking(true)
	defer vc.conn.Speaking(false)

	start := time.Now()
	for {
		select {
		case <-vc.close:
			fmt.Println("Voice connection closed")
			return
		case ctrl := <-vc.control:
			switch ctrl {
			case Stop:
				vc.control <- Pause
				return
			case Skip:
				return
			case Pause:
				done := false
				for {
					ctl, ok := <-vc.control
					if !ok {
						return
					}

					switch ctl {
					case Skip:
						return
					case Resume:
						done = true
					}
					if done {
						break
					}
				}
			}
		default:
		}

		err = binary.Read(dcabuf, binary.LittleEndian, &opusLen)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Println("dcabuf, &opusLen: EOF")
			return
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		opus := make([]byte, opusLen)
		err = binary.Read(dcabuf, binary.LittleEndian, &opus)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Println("dcabuf, &opus: EOF")
			return
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		vc.conn.Send <- opus
		if vc.current != nil {
			vc.Lock()
			vc.current.Remaining = (vc.current.Duration - int(time.Since(start).Seconds()))
			vc.Unlock()
		}
	}
}

func (mp *MusicPlayer) start(vc *Connection) {
	var i int
	var track *Track

	for {
		select {
		case <-vc.close:
			return
		default:
		}

		if len(vc.Queue) == 0 {
			<-time.Tick(time.Second)
			continue
		}

		vc.Lock()
		if len(vc.Queue)-1 >= i {
			track = vc.Queue[i]
		} else {
			i = 0
			vc.Unlock()
			continue
		}
		vc.Unlock()

		vc.current = track
		mp.play(vc, track)
		vc.current = nil

		vc.Lock()
		if len(vc.Queue) > 0 {
			vc.Queue = vc.Queue[1:]
		}
		vc.Unlock()
	}
}

func (mp *MusicPlayer) gostart(m *dgofw.DiscordMessage) {
	vc, ok := mp.VoiceConnections[m.GuildID()]
	vc.Lock()
	defer vc.Unlock()

	if !ok || vc.close != nil || vc.control != nil {
		return
	}
	vc.close = make(chan struct{}, 1)
	vc.control = make(chan controlMessage)

	go mp.start(vc)
}

func (mp *MusicPlayer) queue(link string, m *dgofw.DiscordMessage) (err error) {
	vc, ok := mp.VoiceConnections[m.GuildID()]
	if !ok {
		return fmt.Errorf("Not in a voice channel")
	}

	if len(vc.Queue) >= vc.MaxQueueSize {
		m.Reply("Can't queue any more tracks right now")
		return
	}
	var cmd *exec.Cmd

	// Had to do this because of where it was hosted
	if runtime.GOOS == "linux" {
		cmd = exec.Command("python3", "youtube-dl", "-i", "-j", "--youtube-skip-dash-manifest", link)
	} else if runtime.GOOS == "windows" {
		cmd = exec.Command("./youtube-dl", "-i", "-j", "--youtube-skip-dash-manifest", link)
	}

	output, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		m.Reply("Failed to add song to playlist")
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		m.Reply("Failed to add song to playlist")
		return
	}
	defer func() {
		go cmd.Wait()
	}()

	scanner := bufio.NewScanner(output)

	s := &Track{}
	for scanner.Scan() {
		err = json.Unmarshal(scanner.Bytes(), &s)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if s.URL == "" {
			s.URL = "https://youtube.com/watch?v=" + s.ID
		}

		s.AddedBy = m.Author
		s.LenMinutes = int(math.Floor(float64(s.Duration) / 60))
		s.LenSeconds = s.Duration - s.LenMinutes*60

		vc.Lock()
		vc.Queue = append(vc.Queue, s)
		vc.Unlock()
	}
	m.Reply(fmt.Sprintf("Added **%s** to the queue.", s.Title))
	return
}

var urlRegex = regexp.MustCompile(`^<?(https?:\/\/)?((www\.)?youtube\.com|youtu\.?be)\/.+>?$`)

func (mp *MusicPlayer) OnMessage(m *dgofw.DiscordMessage) {
	arg1 := m.Arg("arg1")
	if arg1 == "" {
		return
	}

	presences := m.Guild().VoiceStates()

	switch arg1 {
	case "join":
		if _, ok := mp.VoiceConnections[m.GuildID()]; ok {
			m.Reply("I'm already in a voice channel!")
			return
		}

		for _, presence := range presences {
			if presence.UserID == m.Author.ID() {
				vc := mp.discord.NewVoiceConnection(presence.GuildID, presence.ChannelID)
				conn := &Connection{
					Guild:        presence.GuildID,
					Channel:      presence.ChannelID,
					MaxQueueSize: 32,
					Queue:        make([]*Track, 0),
					current:      nil,
					conn:         vc,
				}

				mp.Lock()
				mp.VoiceConnections[presence.GuildID] = conn
				mp.Unlock()
				return
			}
		}

		m.Reply("You're not in a voice channel")
	case "leave":
		vc, ok := mp.VoiceConnections[m.GuildID()]
		if ok && m.IsMod() {
			vc.conn.Leave()
			mp.Lock()
			close(vc.close)
			close(vc.control)
			delete(mp.VoiceConnections, vc.Guild)
			mp.Unlock()
		}
	case "play":
		mp.gostart(m)
		urls := m.Arg("arg2")
		if urlRegex.MatchString(urls) {
			if urls[0] == '<' {
				urls = strings.Trim(urls, "<>")
			}
			url, err := url.Parse(urls)
			if err != nil {
				m.Reply("Invalid url")
				return
			}

			err = mp.queue(url.String(), m)
			if err != nil {
				fmt.Println(err)
				m.Reply("Something happened...")
				return
			}
		} else {
			err := mp.queue("ytsearch:"+urls, m)
			if err != nil {
				fmt.Println(err)
				m.Reply("Something happened...")
				return
			}
		}
	case "pause", "resume", "skip", "stop":
		mp.control(m.GuildID(), arg1)
	case "np", "current":
		if vc, ok := mp.VoiceConnections[m.GuildID()]; ok {
			if vc.current != nil {
				minutes := int(math.Floor(float64(vc.current.Remaining) / 60))
				seconds := vc.current.Remaining - minutes*60
				m.ReplyEmbed(&discordgo.MessageEmbed{
					Author: &discordgo.MessageEmbedAuthor{
						Name:    "Added by " + vc.current.AddedBy.Username(),
						IconURL: vc.current.AddedBy.Avatar(),
					},
					Title: vc.current.Title,
					URL:   vc.current.URL,
					Color: m.Session().State.UserColor(vc.current.AddedBy.ID(), m.ChannelID()),
					Image: &discordgo.MessageEmbedImage{
						URL: vc.current.Thumbnail,
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("Play time: %d:%d / %d:%d", minutes, seconds, vc.current.LenMinutes, vc.current.LenSeconds),
					},
				})
			}
		}
	case "queue", "list":
		if vc, ok := mp.VoiceConnections[m.GuildID()]; ok {
			if len(vc.Queue) > 0 {
				var buf bytes.Buffer
				buf.WriteString(fmt.Sprintf("`Now playing`  **%s** added by **%s**\n\n", vc.Queue[0].Title, vc.Queue[0].AddedBy.Username()))
				for i, track := range vc.Queue[1:] {
					if i == 9 {
						l := len(vc.Queue) - 10
						buf.WriteString(fmt.Sprintf("`10`  **%s** added by **%s**\nAnd %d more", track.Title, track.AddedBy.Username(), l))
						break
					}
					buf.WriteString(fmt.Sprintf("`%d`    **%s** added by **%s**\n", i+1, track.Title, track.AddedBy.Username()))
				}
				m.Reply(buf.String())
			} else {
				m.Reply("Nothing in the queue!")
			}
		}
	case "clear":
		if m.IsMod() {
			if vc, ok := mp.VoiceConnections[m.GuildID()]; ok {
				vc.Lock()
				vc.Queue = []*Track{}
				vc.Unlock()
				m.Reply("Cleared the queue")
			}
		}
	case "shuffle":
		if vc, ok := mp.VoiceConnections[m.GuildID()]; ok {
			m2 := m.Reply("Shuffling...")
			time.Sleep(time.Second)
			sort.Sort(vc.Queue)
			m2.Edit("Shuffled!")
		}
	}
}

func (mp *MusicPlayer) control(guild, ctrl string) {
	if vc, ok := mp.VoiceConnections[guild]; ok {
		switch ctrl {
		case "pause":
			vc.control <- Pause
		case "resume":
			vc.control <- Resume
		case "skip":
			vc.control <- Skip
		case "stop":
			vc.control <- Stop
		}
	}
}
