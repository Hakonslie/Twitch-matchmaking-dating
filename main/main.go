package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nicklaw5/helix"
	"github.com/sirupsen/logrus"

	"twating/v0/config"
	"twating/v0/irc"
)

const size = 123

func main() {

	ctx := context.Background()
	logger := logrus.WithField("bot", "running")
	appConfig := config.OpenConfig(logger)
	env := config.Environment{
		Logger:  logger,
		Context: ctx,
		Config:  appConfig,
	}

	client, err := helix.NewClient(&helix.Options{
		ClientID:        env.Config.Twitch.ClientID,
		ClientSecret:    env.Config.Twitch.ClientSecret,
		UserAccessToken: env.Config.Twitch.Token,
	})
	if err != nil {
		log.Panic(err)
	}

	ic := irc.NewConn("irc.chat.twitch.tv:6667")
	if err := ic.Dial(env); err != nil {
		logger.WithError(err).Error()
		return
	} else {
		fmt.Printf("Connnected to channel: %s \n", env.Config.Irc.Channel)
	}

	approved := make(map[string][]string)
	ignored := make(map[string]bool)

	r := make(chan []string)
	go loop(ic, r)
	for {
		select {
		/*
			case msg := <-r:
				fmt.Printf("\nGot message: %s from user %s \n", msg[0], msg[1])
				switch msg[0] {
				case "!Join":
					if _, ok := approved[msg[1]]; !ok {
						if len(approved) < 10 {
							handleMessage(client, msg[1], approved)
							ic.SendRaw(fmt.Sprintf("PRIVMSG %s : You're in %s!", env.Config.Irc.Channel, msg[1]))
						} else {
							ic.SendRaw(fmt.Sprintf("PRIVMSG %s : The pool is full %s! You can use !Match to rest", env.Config.Irc.Channel, msg[1]))
						}
					}
				case "!Match":
					if len(approved) > 5 {
						a, b, p := calculateBestMatch(logger, approved)
						ic.SendRaw(fmt.Sprintf("PRIVMSG %s : I think %s and %s should chat it up! (%d%% accuracy)", env.Config.Irc.Channel, a, b, p))
						approved = make(map[string][]string)
					} else {
						ic.SendRaw(fmt.Sprintf("PRIVMSG %s :%s", env.Config.Irc.Channel, "Would like at least 5 participants :("))
					}
				}
		*/
		case msg := <-r:
			if msg[1] != "" {
				if _, ok := ignored[msg[1]]; !ok {
					if _, ok := approved[msg[1]]; !ok {
						if len(approved) < size {
							if handleMessage(client, msg[1], approved, ignored) {
								fmt.Printf("%d/%d User admitted: %s \n", len(approved), size, msg[1])
							}
						} else {
							a, b, p := calculateBestMatch(logger, approved)
							fmt.Printf("\n---------------------------------------------------------\nRESULTS:  %s and %s are the best match, they should chat it up!\nAccuracy: %d%%\n\n", a, b, p)

							approved = make(map[string][]string)
						}
					}
				}

			}
		}
	}
}

func calculateBestMatch(logger logrus.FieldLogger, users map[string][]string) (string, string, int) {
	matchMap := make(map[string]map[string]int)
	for user1, v := range users {
		matchMap[user1] = make(map[string]int)
		for _, p := range v {
		user2Loop:
			for user2, j := range users {
				if user1 == user2 {
					continue user2Loop
				}
				matches := 0
				for _, i := range j {
					if i == p {
						matches++
					}
				}
				matchMap[user1][user2] += int((float64(matches) / float64(len(j))) * 100.0)
			}
		}
	}
	logger.WithField("map", matchMap).Info()
	type couple struct {
		i, k string
	}
	coupleMap := make(map[couple]int)
	for user1, v := range matchMap {
		for user2, matchPer := range v {
			// Doesn't have a similar match from before
			if _, ok := coupleMap[couple{i: user2, k: user1}]; !ok {
				coupleMap[couple{
					i: user1,
					k: user2,
				}] = (matchPer + matchMap[user2][user1]) / 2
			}
		}
	}
	logger.WithField("map", coupleMap).Info()
	var h couple
	hValue := 0
	for k, v := range coupleMap {
		if v > hValue {
			hValue = v
			h = k
		}
	}

	return h.i, h.k, hValue
}

func handleMessage(client *helix.Client, user string, users map[string][]string, ignore map[string]bool) bool {
	id, err := getUserID(client, user)
	if err != nil {
		return false
	}

	par := helix.UsersFollowsParams{FromID: id, First: 100}
	follows, err := client.GetUsersFollows(&par)
	if err != nil {
		return false
	} else if len(follows.Data.Follows) < 10 {
		ignore[user] = true
		return false
	}
	var uf []string
	for _, f := range follows.Data.Follows {
		uf = append(uf, f.ToID)
	}
	users[user] = uf
	return true
}

func getUserID(tc *helix.Client, name string) (string, error) {
	user := helix.UsersParams{Logins: []string{name}}
	users, err := tc.GetUsers(&user)
	if err != nil {
		return "", err
	}
	return users.Data.Users[0].ID, nil
}

func loop(ic *irc.Conn, r chan []string) {
	for {
		select {
		// Add messages to pending slice when they arrive from IRC channel
		case message := <-ic.Out:
			r <- []string{message.Trailing(), message.User}
		}
	}
}
