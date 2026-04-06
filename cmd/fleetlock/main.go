package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	fleetlock "github.com/poseidon/fleetlock/internal"
)

var (
	// version provided by compile time -ldflags
	version = "was not built properly"
	// logger defaults to info logging
	log = logrus.New()
)

func main() {
	flags := struct {
		address                string
		logLevel               string
		drainMaxWait           time.Duration
		maintenanceWindowStart string
		maintenanceWindowEnd   string
		slackBotToken          string
		slackChannelID         string
		version                bool
		help                   bool
	}{}

	flag.StringVar(&flags.address, "address", "0.0.0.0:8080", "HTTP listen address")
	// log levels https://github.com/sirupsen/logrus/blob/master/logrus.go#L36
	flag.StringVar(&flags.logLevel, "log-level", "info", "Set the logging level")
	flag.DurationVar(&flags.drainMaxWait, "drain-max-wait", 60*time.Second, "Maximum time to wait for pod evictions during drain")
	flag.StringVar(&flags.maintenanceWindowStart, "maintenance-window-start", "", "Start of maintenance window in HH:MM format (UTC)")
	flag.StringVar(&flags.maintenanceWindowEnd, "maintenance-window-end", "", "End of maintenance window in HH:MM format (UTC)")
	flag.StringVar(&flags.slackBotToken, "slack-bot-token", "", "Slack Bot User OAuth Token for notifications")
	flag.StringVar(&flags.slackChannelID, "slack-channel-id", "", "Slack channel ID for lock/unlock notifications")
	// subcommands
	flag.BoolVar(&flags.version, "version", false, "Print version and exit")
	flag.BoolVar(&flags.help, "help", false, "Print usage and exit")

	// parse command line arguments
	flag.Parse()

	if flags.version {
		fmt.Println(version)
		return
	}

	if flags.help {
		flag.Usage()
		return
	}

	// logger
	lvl, err := logrus.ParseLevel(flags.logLevel)
	if err != nil {
		log.Fatalf("invalid log-level: %v", err)
	}
	log.Level = lvl

	// HTTP Server
	config := &fleetlock.Config{
		Logger:                 log,
		DrainMaxWait:           flags.drainMaxWait,
		MaintenanceWindowStart: flags.maintenanceWindowStart,
		MaintenanceWindowEnd:   flags.maintenanceWindowEnd,
		SlackBotToken:          flags.slackBotToken,
		SlackChannelID:         flags.slackChannelID,
	}
	server, err := fleetlock.NewServer(config)
	if err != nil {
		log.Fatalf("main: NewServer error %v", err)
	}

	log.Infof("main: starting fleetlock on %s", flags.address)
	err = http.ListenAndServe(flags.address, server)
	if err != nil {
		log.Fatalf("main: ListenAndServe error: %v", err)
	}
}
