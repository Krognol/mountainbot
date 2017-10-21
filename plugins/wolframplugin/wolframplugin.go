package wolframplugin

import (
	"fmt"

	"github.com/Krognol/dgofw"
	"github.com/Krognol/go-wolfram"
)

type Wap struct {
	Client *wolfram.Client
}

func NewWolframPlugin(appid string) *Wap {
	return &Wap{
		Client: &wolfram.Client{
			AppID: appid,
		},
	}
}

func (w *Wap) OnMessage(m *dgofw.DiscordMessage) {
	query := m.Arg("query")
	if query == "" {
		return
	}

	res, err := w.Client.GetShortAnswerQuery(query, wolfram.None, 0)
	if err != nil {
		m.Reply("Something happened...")
		fmt.Println(err)
		return
	}

	m.Reply(res)
}
