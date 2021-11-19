package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ha "github.com/beanz/homeassistant-go/pkg/types"
	"github.com/beanz/rrf-go/pkg/netrrf"
	"github.com/beanz/rrf-go/pkg/types"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

type Msg struct {
	topic  string
	body   interface{}
	retain bool
}

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

	bridgeAvailabilityTopic := AvailabilityTopic(cfg, "bridge")

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

	err = cm.AwaitConnection(ctx)
	if err != nil { // Should only happen when context is cancelled
		return fmt.Errorf("broker connection error: %s", err)
	}

	msgc := make(chan *Msg, 100)

	for i := range cfg.Devices {
		go pollDevice(ctx, cfg.Devices[i], cfg, msgc, logger)
	}

LOOP:
	for {
		err = cm.AwaitConnection(ctx)
		if err != nil { // Should only happen when context is cancelled
			return fmt.Errorf("broker connection error: %s", err)
		}

		select {
		case m := <-msgc:
			err = pub(ctx, cm, m.topic, m.body, m.retain)
			if err != nil {
				logger.Printf(
					"failed to publish discovery message for %s: %s",
					m.topic, err)
			}
		case <-sigc:
			break LOOP
		}
	}
	logger.Println("shutting down")

	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pr, err := cm.Publish(ctx, &paho.Publish{
		QoS:     1,
		Topic:   bridgeAvailabilityTopic,
		Payload: []byte("offline"),
		Retain:  true,
	})
	if err != nil {
		fmt.Printf("failed to publish availability offline message: %s\n", err)
	} else if pr.ReasonCode != 0 && pr.ReasonCode != 16 {
		// 16 = Server received message but there are no subscribers
		fmt.Printf("publish availability offline reason code %dn",
			pr.ReasonCode)
	}
	_ = cm.Disconnect(ctx)

	return nil
}

type PollResult struct {
	Host              string
	TopicFriendlyName string
	AvailabilityTopic string
	StateTopic        string
	Config            *types.ConfigResponse
	Status2           *types.StatusResponse
	Status3           *types.StatusResponse
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

func pollDevice(ctx context.Context, host string, cfg *Config, msgc chan *Msg, logger *log.Logger) {
	ticker := time.NewTicker(cfg.Interval)

	availabilityTopic := AvailabilityTopic(cfg, topicSafe(host))

	var lastDiscovery *time.Time
	lastAvailability := ""
	for {
		newAvailability := "offline"
		logger.Printf("%s tick\n", host)
		rrf := netrrf.NewClient(host, cfg.Password)
		now := time.Now()
		var cr *types.ConfigResponse
		var r *PollResult
		var err error
		if lastDiscovery == nil || (*lastDiscovery).Add(cfg.DiscoveryInterval).Before(now) {
			cr, err = rrf.Config(ctx)
			if err != nil {
				logger.Printf("poll of %s config failed: %v\n", host, err)
				goto TICK
			}
			lastDiscovery = &now
		}

		{
			s2, err := rrf.Status(ctx, 2)
			if err != nil {
				logger.Printf("poll of %s status2 failed: %v\n", host, err)
				goto TICK
			}
			{
				s3, err := rrf.Status(ctx, 3)
				if err != nil {
					logger.Printf("poll of %s status3 failed: %v\n", host, err)
					goto TICK
				}
				newAvailability = "online"
				name := topicSafe(s2.Name)
				r = &PollResult{
					Host:              host,
					TopicFriendlyName: name,
					AvailabilityTopic: availabilityTopic,
					StateTopic:        StateTopic(cfg, name),
					Config:            cr,
					Status2:           s2,
					Status3:           s3,
				}
			}
		}

	TICK:

		if lastAvailability != newAvailability {
			lastAvailability = newAvailability
			msgc <- &Msg{topic: availabilityTopic, body: newAvailability, retain: true}
		}

		if r != nil {
			logger.Printf("got results for %s (name=%s)\n",
				r.Host, r.Status2.Name)
			variables := variablesFromResults(r)
			if r.Config != nil {
				msgs := discoveryMessages(cfg, r, variables)
				for _, msg := range msgs {
					msgc <- msg
				}
			}
			msg := resultMessage(r, variables)
			msgc <- msg
		}

		<-ticker.C
	}
}

func topicSafe(s string) string {
	r := strings.ReplaceAll(s, "/", "_slash_")
	r = strings.ReplaceAll(r, "#", "_hash_")
	r = strings.ReplaceAll(r, "+", "_plus_")
	r = strings.ReplaceAll(r, "-", "_")
	r = strings.ReplaceAll(r, ":", "_")
	r = strings.TrimLeft(r, "_")
	r = strings.TrimRight(r, "_")
	return strings.ToLower(r)
}

func ConfigTopic(cfg *Config, name, variable string) string {
	return fmt.Sprintf("%s/sensor/%s_%s/config",
		cfg.DiscoveryTopicPrefix, name, variable)
}

func StateTopic(cfg *Config, name string) string {
	return fmt.Sprintf("%s/%s/state", cfg.TopicPrefix, name)
}

func AvailabilityTopic(cfg *Config, name string) string {
	return fmt.Sprintf("%s/%s/availability", cfg.TopicPrefix, name)
}

type Variable struct {
	field       string
	icon        string
	units       string
	deviceClass *ha.DeviceClass
	value       interface{}
}

func variablesFromResults(res *PollResult) []Variable {
	dcTemp := ha.DeviceClassTemperature
	dcVolt := ha.DeviceClassVoltage

	// something graphable:
	// 0 for off, 1 for idle, 2 for exception and 3 for printing
	stateCode := map[types.Status]int{
		types.Configuring:  1,
		types.Idle:         1,
		types.Busy:         1,
		types.Printing:     3,
		types.Pausing:      3,
		types.Stopped:      2,
		types.Resuming:     3,
		types.Halted:       2,
		types.Flashing:     2,
		types.ToolChanging: 3,
	}

	variables := []Variable{
		{
			field: "state",
			value: res.Status2.Status.String(),
		},
		{
			field: "state_code",
			value: stateCode[res.Status2.Status],
		},
		{
			field: "file_time_remaining",
			value: res.Status3.TimesLeft.File,
		},
		{
			field: "filament_time_remaining",
			value: res.Status3.TimesLeft.Filament,
		},
		{
			field: "layer_time_remaining",
			value: res.Status3.TimesLeft.Layer,
		},
		{
			field:       "mcu_temp_min",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status2.MCUTemp.Min,
		},
		{
			field:       "mcu_temp_cur",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status2.MCUTemp.Cur,
		},
		{
			field:       "mcu_temp_max",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status2.MCUTemp.Max,
		},
		{
			field:       "vin_min",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status2.VIN.Min,
		},
		{
			field:       "vin_cur",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status2.VIN.Cur,
		},
		{
			field:       "vin_max",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status2.VIN.Max,
		},
		{
			field: "geometry",
			value: res.Status2.Geometry,
		},
		{
			field: "layer",
			value: res.Status3.CurrentLayer,
		},
	}
	if len(res.Status2.Coordinates.XYZ) == 3 {
		for i, v := range []string{"x", "y", "z"} {
			variables = append(variables, Variable{
				field: v,
				icon:  "mdi:axis-" + v + "-arrow",
				value: res.Status2.Coordinates.XYZ[i],
			})
		}
	}
	for i := range res.Status2.Coordinates.Extruder {
		variables = append(variables, Variable{
			field: fmt.Sprintf("e%d", i),
			icon:  "mdi:mdi-printer-3d-nozzle",
			value: res.Status2.Coordinates.Extruder[i],
		})
	}
	for i := range res.Status2.Temps.Current {
		if res.Status2.Temps.Current[i] > 1000 {
			continue
		}
		temp := fmt.Sprintf("temp%d", i)
		if len(res.Status2.Temps.Names) > i && res.Status2.Temps.Names[i] != "" {
			temp = res.Status2.Temps.Names[i]
		}
		variables = append(variables, Variable{
			field:       temp,
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status2.Temps.Current[i],
		})
	}
	return variables
}

func discoveryMessages(cfg *Config, res *PollResult, variables []Variable) []*Msg {
	availability := []ha.Availability{
		{Topic: AvailabilityTopic(cfg, "bridge")},
		{Topic: res.AvailabilityTopic},
	}
	realName := res.Status2.Name

	msgs := []*Msg{}
	for _, v := range variables {
		sensor := ha.Sensor{
			Availability: availability,
			Name:         realName + " " + v.field,
			Icon:         "mdi:printer-3d",
			UniqueID:     res.TopicFriendlyName + "_" + v.field,
			Device: ha.Device{
				Identifiers: []string{
					res.TopicFriendlyName,
					res.TopicFriendlyName + "_" + v.field,
				},
				ConfigurationURL: "http://" + res.Host,
				Name:             realName,
				SwVersion:        res.Config.FirmwareName + " v" + res.Config.FirmwareVersion + " (" + string(res.Config.FirmwareDate) + ")",
				Model:            res.Config.FirmwareElectronics,
			},
			StateTopic:    res.StateTopic,
			ValueTemplate: "{{ value_json." + v.field + "}}",
		}
		if v.units != "" {
			sensor.UnitOfMeasurement = v.units
		}
		if v.deviceClass != nil {
			sensor.DeviceClass = *v.deviceClass
		}
		if v.icon != "" {
			sensor.Icon = v.icon
		}
		msgs = append(msgs, &Msg{
			topic:  ConfigTopic(cfg, res.TopicFriendlyName, v.field),
			body:   sensor,
			retain: true,
		})
	}
	return msgs
}

func resultMessage(res *PollResult, variables []Variable) *Msg {
	t := float64(time.Now().UnixNano()/1000000) / 1000
	msg := map[string]interface{}{
		"t": t,
	}
	for _, v := range variables {
		msg[v.field] = v.value
	}
	return &Msg{topic: res.StateTopic, body: msg, retain: false}
}
