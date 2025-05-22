package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	Token        = readToken("token.txt")
	ChannelID    = "1355803790774767646"
	GuildID      = "1353806073999396986"
	statsMessage *discordgo.Message
	botStartTime = time.Now()
	botUserID    string
)

func readToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("Could not read token.txt")
	}
	return strings.TrimSpace(string(data))
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		panic("Error creating Discord session: " + err.Error())
	}
	dg.Identify.Intents = discordgo.IntentsGuilds

	dg.AddHandlerOnce(onReady)

	err = dg.Open()
	if err != nil {
		panic("Error opening connection: " + err.Error())
	}

	fmt.Println("Bot is running.")
	select {}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	botUserID = r.User.ID
	fmt.Printf("Logged in as: %s#%s\n", r.User.Username, r.User.Discriminator)

	// Attempt to find existing stats message
	messages, err := s.ChannelMessages(ChannelID, 10, "", "", "")
	if err == nil {
		for _, msg := range messages {
			if msg.Author != nil && msg.Author.ID == botUserID {
				statsMessage = msg
				fmt.Println("Reusing previous stats message:", msg.ID)
				break
			}
		}
	} else {
		fmt.Println("Failed to get channel messages:", err)
	}

	go statsLoop(s)
}

func statsLoop(s *discordgo.Session) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			embed := buildStatsEmbed()
			if statsMessage == nil {
				msg, err := s.ChannelMessageSendEmbed(ChannelID, embed)
				if err == nil {
					statsMessage = msg
					fmt.Println("Created new stats message:", msg.ID)
				} else {
					fmt.Println("Send error:", err)
				}
			} else {
				_, err := s.ChannelMessageEditEmbed(ChannelID, statsMessage.ID, embed)
				if err != nil {
					fmt.Println("Edit error:", err)
					statsMessage = nil // fallback to resend if message is deleted
				}
			}
		}
	}
}

func buildStatsEmbed() *discordgo.MessageEmbed {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Percent(0, false)
	d, _ := disk.Usage("/")
	uptime := time.Since(botStartTime)

	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60
	uptimeStr := fmt.Sprintf("%d days, %d hours, %d minutes, and %d seconds", days, hours, minutes, seconds)

	embed := &discordgo.MessageEmbed{
		Title:       "Bot stats",
		Description: "System stats updated every 5 seconds",
		Color:       0x00ffcc,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "RAM",
				Value:  fmt.Sprintf("Used: %.2fMB / %.2fMB", float64(v.Used)/1024/1024, float64(v.Total)/1024/1024),
				Inline: false,
			},
			{
				Name:   "CPU",
				Value:  fmt.Sprintf("%.1f%% (%d core/s)", c[0], runtime.NumCPU()),
				Inline: false,
			},
			{
				Name:   "Disk",
				Value:  fmt.Sprintf("Used: %.2fGB / %.2fGB", float64(d.Used)/1024/1024/1024, float64(d.Total)/1024/1024/1024),
				Inline: false,
			},
			{
				Name:   "OS",
				Value:  fmt.Sprintf("Running on %s", runtime.GOOS),
				Inline: true,
			},
			{
				Name:   "Go",
				Value:  fmt.Sprintf("Using Go `%s`", runtime.Version()),
				Inline: true,
			},
			{
				Name:   "Uptime",
				Value:  fmt.Sprintf("I have been running for %s", uptimeStr),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}
