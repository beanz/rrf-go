package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func Run(cfg *Config, logger *log.Logger) error {
	logger.Printf("%s v%s\n", cfg.AppName, cfg.Version)
	logger.Println("Starting Home Assistant integration")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	signal.Notify(sigc, syscall.SIGTERM)
	brokerURL, err := url.Parse(cfg.Broker)
	if err != nil {
		return fmt.Errorf("invalid broker url: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridgeAvailabilityTopic := cfg.TopicPrefix + "/bridge/availability"

	cmCfg := autopaho.ClientConfig{
		BrokerUrls:        []*url.URL{brokerURL},
		KeepAlive:         uint16(cfg.KeepAlive),
		ConnectRetryDelay: cfg.ConnectRetryDelay,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			logger.Println("MQTT connection up")
			err = pub(ctx, cm, bridgeAvailabilityTopic, "online", true)
			if err != nil {
				logger.Printf(
					"failed to publish availability online message: %s", err)
			}
		},
		OnConnectError: func(err error) {
			logger.Printf("error whilst attempting connection: %s\n", err)
		},
		Debug: paho.NOOPLogger{},
		ClientConfig: paho.ClientConfig{
			ClientID: cfg.ClientID,
			OnClientError: func(err error) {
				logger.Printf("server requested disconnect: %s\n", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					logger.Printf("server requested disconnect: %s\n",
						d.Properties.ReasonString)
				} else {
					logger.Printf(
						"server requested disconnect; reason code: %d\n",
						d.ReasonCode)
				}
			},
		},
	}
	logger.Printf("setting will message %s: %s\n",
		bridgeAvailabilityTopic, "offline")
	cmCfg.SetWillMessage(bridgeAvailabilityTopic, []byte("offline"), 1, true)

	cm, err := autopaho.NewConnection(ctx, cmCfg)
	if err != nil {
		return err
	}
LOOP:
	for {
		err = cm.AwaitConnection(ctx)
		if err != nil { // Should only happen when context is cancelled
			return fmt.Errorf("broker connection error: %s", err)
		}

		select {
		case <-sigc:
			break LOOP
		}
	}
	logger.Println("shutting down")

	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = pub(ctx, cm, bridgeAvailabilityTopic, "offline", true)
	if err != nil {
		logger.Printf(
			"failed to publish availability online message: %s", err)
	}
	_ = cm.Disconnect(ctx)

	return nil
}

func pub(ctx context.Context, cm *autopaho.ConnectionManager, topic string, body interface{}, retain bool) error {
	var b []byte
	var err error
	if s, ok := body.(string); ok {
		b = []byte(s)
	} else {
		b, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	go func(msg []byte) {
		pr, err := cm.Publish(ctx, &paho.Publish{
			QoS:     1,
			Topic:   topic,
			Payload: msg,
			Retain:  retain,
		})
		if err != nil {
			fmt.Printf("error publishing: %s\n", err)
		} else if pr.ReasonCode != 0 && pr.ReasonCode != 16 {
			// 16 = Server received message but there are no subscribers
			fmt.Printf("reason code %d received\n", pr.ReasonCode)
		}
	}(b)
	return nil
}
