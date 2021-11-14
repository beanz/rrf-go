/*
Copyright © 2021 Mark Hindess

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beanz/rrf-go/pkg/ha"
	"github.com/beanz/rrf-go/pkg/netrrf"
	"github.com/urfave/cli/v2"
)

// Version is overridden at build time
var Version = "0.0.0+Dev"

const appName = "rrf-go"

func main() {
	stdout := os.Stdout

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "password for the rrf device(s)",
				EnvVars: []string{"RRF_PASSWORD"},
			},
		},

		Commands: []*cli.Command{
			{
				Name:    "info",
				Aliases: []string{"i"},
				Usage:   "fetch basic information about reprapfirmware device(s)",
				Action: func(c *cli.Context) error {
					pw := c.String("password")
					ctx := context.Background()
					for _, h := range c.Args().Slice() {
						rrf := netrrf.NewClient(h, pw)
						cfg, err := rrf.Config(ctx)
						if err != nil {
							return err
						}
						s2, err := rrf.Status2(ctx)
						if err != nil {
							return err
						}
						if c.String("output") == "json" {
							pj, err := json.MarshalIndent(
								map[string]interface{}{
									"config":  cfg,
									"status2": s2,
								}, "", "  ")
							if err != nil {
								return err
							}
							fmt.Fprintf(stdout, string(pj))
							return nil
						}
						fmt.Fprintf(stdout, `%s:
  Name: %s
  State: %s
  Firmware: %s v%s (%s)
  Electronics: %s
  Geometry: %s
`,
							h, s2.Name, s2.Status,
							cfg.FirmwareName,
							cfg.FirmwareVersion, cfg.FirmwareDate,
							cfg.FirmwareElectronics,
							s2.Geometry,
						)
						for i := 0; i < s2.Axes; i++ {
							var homed string
							if !s2.Coordinates.AxisHomed[i] {
								homed = " (not homed)"
							}
							fmt.Fprintf(stdout,
								"  Axis %d: %-7.2f (min=%.2f max=%.2f)%s\n",
								i, s2.Coordinates.XYZ[i],
								cfg.AxisMins[i], cfg.AxisMaxes[i], homed)
						}
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output format",
					},
				},
			},
			{
				Name:    "homeassistant",
				Aliases: []string{"ha"},
				Usage:   "homeassistant integration",
				Action: func(c *cli.Context) error {
					fmt.Println("homeassistant task: ", c.Args().First())
					cfg := &ha.Config{
						AppName:              appName,
						Version:              Version,
						Devices:              c.Args().Slice(),
						Broker:               c.String("broker"),
						ClientID:             c.String("client-id"),
						TopicPrefix:          c.String("topic-prefix"),
						DiscoveryTopicPrefix: c.String("discovery-topic-prefix"),
						Interval:             c.Duration("interval"),
						DiscoveryInterval:    c.Duration("discovery-interval"),
						ConnectRetryDelay:    c.Duration("connect-retry-delay"),
						KeepAlive:            c.Int("keepalive"),
					}
					return ha.Run(
						cfg,
						log.New(stdout, "",
							log.Ldate|log.Ltime|log.Lmicroseconds))
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "broker",
						Usage:   "MQTT broker to connect to",
						EnvVars: []string{"RRF_BROKER"},
					},
					&cli.StringFlag{
						Name:    "client-id",
						Aliases: []string{"cid"},
						Usage:   "MQTT broker to connect to",
						EnvVars: []string{"RRF_CLIENT_ID"},
						Value:   appName,
					},
					&cli.StringFlag{
						Name:    "topic-prefix",
						Aliases: []string{"at"},
						Usage:   "MQTT topic prefix for published data, availability, etc (default rrf-go)",
						EnvVars: []string{"RRF_TOPIC_PREFIX"},
						Value:   appName,
					},
					&cli.StringFlag{
						Name:    "discovery-topic-prefix",
						Aliases: []string{"dp"},
						Usage:   "MQTT topic prefix for discovery (default 'homeassistant')",
						EnvVars: []string{"RRF_DISCOVERY_TOPIC_PREFIX"},
						Value:   "bar", // todo change default to homeassistant
					},
					&cli.DurationFlag{
						Name: "interval", Aliases: []string{"i"},
						Usage: "interval between polling devices",
						Value: time.Second * 60,
					},
					&cli.DurationFlag{
						Name: "discovery-interval", Aliases: []string{"di"},
						Usage: "interval between publishing discovery messages",
						Value: time.Hour,
					},
					&cli.DurationFlag{
						Name:  "connect-retry-delay",
						Usage: "interval between broker reconnection attempts",
						Value: time.Second * 10,
					},
					&cli.IntFlag{
						Name:  "keepalive",
						Usage: "MQTT keepalive parameter",
						Value: 30,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
