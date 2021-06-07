package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// endpoint specifies the URL to the corona-zahlen district API endpoint.
// See https://api.corona-zahlen.org/docs/endpoints/districts.html#districts-2.
const endpoint = "https://api.corona-zahlen.org/districts/%s"

// key specifies the district key of Miltenberg, used for the endpoint.
const key = "09676"

// timeFmt is the format used to display the date of a data fetch.
const timeFmt = "02.01.2006"

// Response implements a typical response from the district endpoint, see endpoint.
type Response struct {
	Data map[string]struct {
		AGS           string  `json:"ags"`
		Name          string  `json:"name"`
		County        string  `json:"county"`
		Population    uint32  `json:"population"`
		Cases         uint32  `json:"cases"`
		Deaths        uint32  `json:"deaths"`
		CasesPerWeek  uint32  `json:"casesPerWeek"`
		DeathsPerWeek uint32  `json:"deathsPerWeek"`
		Recovered     uint32  `json:"recovered"`
		WeekIncidence float64 `json:"weekIncidence"`
		CasesPer100k  float64 `json:"casesPer100k"`
		Delta         struct {
			Cases     uint32 `json:"cases"`
			Deaths    uint32 `json:"deaths"`
			Recovered uint32 `json:"recovered"`
		} `json:"delta"`
	} `json:"data"`
	Meta struct {
		Source               string `json:"source"`
		Contact              string `json:"contact"`
		Info                 string `json:"info"`
		LastUpdate           string `json:"lastUpdate"`
		LastCheckedForUpdate string `json:"lastCheckedForUpdate"`
	} `json:"meta"`
}

// Config holds some user-defined values.
type Config struct {
	Timer     string
	Token     string
	ChannelID string
}

// config is an instance of Config used across the application.
var config Config

func init() {
	timer := flag.String("timer", "0 18 * * *", "Specifies the cron notation")
	token := flag.String("token", "", "Specifies the Discord bot token")
	channelID := flag.String("channel", "", "Specifies the Discord channel to use")

	config = Config{
		Timer:     *timer,
		Token:     *token,
		ChannelID: *channelID,
	}
}

func main() {
	client := &http.Client{}
	ctab := cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)))

	discord, err := newDiscordSession(config.Token)
	if err != nil {
		panic(err)
	}

	if _, err := ctab.AddFunc(config.Timer, func() {
		data, err := fetchData(client)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not fetch COVID data: %s\n", err)
			return
		}

		timestamp := getCurrentTimestamp()
		incidence := uint32(data.Data[key].WeekIncidence)

		// Too lazy to handle errors but not too lazy to write this comment which probably took more
		// time than adding a proper error check.
		_, _ = discord.ChannelMessageSend(config.ChannelID, fmt.Sprintf("Inzidenzwert für den %s: **%d**", timestamp, incidence))
		if incidence >= 165 {
			mention, _ := everyone(discord)
			_, _ = discord.ChannelMessageSend(config.ChannelID, fmt.Sprintf("%s Distanzunterricht, wooow", mention))
		}
	}); err != nil {
		panic(err)
	}

	ctab.Start()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

// everyone returns the mention for everyone.
func everyone(discord *discordgo.Session) (string, error) {
	c, err := discord.Channel(config.ChannelID)
	if err != nil {
		return "", err
	}

	g, err := discord.Guild(c.GuildID)
	if err != nil {
		return "", err
	}

	for _, role := range g.Roles {
		if role.Name == "everyone" {
			return role.Mention(), nil
		}
	}

	return "", errors.New("could not find everyone role, what the fuck")
}

// getCurrentTimestamp returns the current time, formatted using timeFmt.
func getCurrentTimestamp() string {
	return time.Now().Format(timeFmt)
}

// newDiscordSession returns a Discord session authenticated using the token.
func newDiscordSession(token string) (*discordgo.Session, error) {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return discord, discord.Open()
}

// fetchData sends a GET request to endpoint for the district identified by key and parses it.
func fetchData(client *http.Client) (*Response, error) {
	resp, err := client.Get(fmt.Sprintf(endpoint, key))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	r := &Response{}
	return r, json.Unmarshal(bytes, r)
}
