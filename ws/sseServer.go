package ws

import (
	"time"

	"github.com/r3labs/sse/v2"
)

var Server *sse.Server

func InitSSE() {
	Server = sse.New()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			Server.Publish("session", &sse.Event{
				Data: []byte("logout"),
			})
		}
	}()
}
