package irc

import (
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/sorcix/irc.v2"

	"twating/v0/config"
)

type Conn struct {
	Host string
	Conn *irc.Conn
	Inp  chan irc.Message
	Out  chan irc.Message
}

func NewConn(host string) *Conn {
	ic := new(Conn)
	ic.Host = host
	ic.Inp = make(chan irc.Message, 100)
	ic.Out = make(chan irc.Message, 100)
	return ic
}

func (ic *Conn) Dial(env config.Environment) error {
	conn, err := irc.Dial(ic.Host)
	if err != nil {
		return err
	}
	ic.Conn = conn

	go ic.srvRecv(env.Logger)
	go ic.srvSend(env.Logger)

	var onJoinMessages []irc.Message
	onJoinMessages = append(onJoinMessages,
		irc.Message{
		Command: irc.PASS,
		Params:  []string{env.Config.Irc.Pass},
	}, irc.Message{
		Command: irc.NICK,
		Params:  []string{env.Config.Irc.Nick},
	}, irc.Message{
		Command: irc.JOIN,
		Params: []string{env.Config.Irc.Channel},
	})
	for _, msg := range onJoinMessages {
		ic.Inp <- msg
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (ic *Conn) srvRecv(logger logrus.FieldLogger) {
	for {
		msg, err := ic.Conn.Decode()
		if err != nil {
			logger.WithError(err).Error()
		}

		if msg.Command == irc.PING {
			ic.Inp <- irc.Message{
				Command: irc.PONG,
				Params:  []string{msg.Trailing()},
			}
		} else {
			ic.Out <- *msg
		}
	}
}

func (ic *Conn) srvSend(logger logrus.FieldLogger) {
	burstLimiter := make(chan time.Time, 5)
	go func() {
		for t := range time.Tick(time.Second * 2) {
			burstLimiter <- t
		}
	}()

	for msg := range ic.Inp {
		<-burstLimiter
		if err := ic.Conn.Encode(&msg); err != nil {
			logger.WithError(err).Error()
		}
	}
}

func (ic *Conn) SendRaw(raw string) {
	ic.Inp <- *irc.ParseMessage(raw)
}