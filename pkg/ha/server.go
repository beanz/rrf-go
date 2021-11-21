package ha

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/beanz/homeassistant-go/pkg/mqtt"
	"github.com/beanz/rrf-go/pkg/netrrf"
	"github.com/beanz/rrf-go/pkg/types"

	ha "github.com/beanz/homeassistant-go/pkg/types"
)

func Run(ctx context.Context, cfg *Config, logger *log.Logger, mqttc mqtt.PubSubServer, msgp, msgs chan *mqtt.Msg) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := range cfg.Devices {
		go deviceLoop(childCtx, cfg.Devices[i], cfg, msgp, logger)
	}

	return mqttc.Run(childCtx, msgp, msgs)
}

type PollResult struct {
	Host              string
	TopicFriendlyName string
	AvailabilityTopic string
	StateTopic        string
	Config            *types.ConfigResponse
	Status            *types.StatusResponse
}

func deviceLoop(ctx context.Context, host string, cfg *Config, msgc chan *mqtt.Msg, logger *log.Logger) {
	ticker := time.NewTicker(cfg.Interval)

	availabilityTopic := AvailabilityTopic(cfg, topicSafe(host))

	var lastDiscovery *time.Time
	lastAvailability := ""
	for {
		newAvailability := "offline"
		if cfg.Debug {
			logger.Printf("%s tick\n", host)
		}
		now := time.Now()
		needsDiscovery := lastDiscovery == nil || (*lastDiscovery).Add(cfg.DiscoveryInterval).Before(now)
		r, err := pollDevice(ctx, host, cfg, needsDiscovery)
		if r != nil {
			newAvailability = "online"
		}
		if lastAvailability != newAvailability {
			if err != nil {
				logger.Printf("poll error: %s\n", err)
			}
			lastAvailability = newAvailability
			msgc <- &mqtt.Msg{Topic: availabilityTopic, Body: newAvailability, Retain: true}
		}
		if r != nil {
			r.AvailabilityTopic = availabilityTopic
			if r.Config != nil {
				lastDiscovery = &now
			}
			if cfg.Debug {
				logger.Printf("got results for %s (name=%s)\n",
					r.Host, r.Status.Name)
			}
			variables := variablesFromResults(r)
			if r.Config != nil {
				msgs := discoveryMessages(cfg, r, variables)
				for _, msg := range msgs {
					msgc <- msg
				}
			}
			msg := resultMessage(r, now, variables)
			msgc <- msg
		}

		<-ticker.C
	}
}

func pollDevice(ctx context.Context, host string, cfg *Config, needsDiscovery bool) (*PollResult, error) {
	rrf := netrrf.NewClient(host, cfg.Password)
	var cr *types.ConfigResponse
	var err error
	if needsDiscovery {
		cr, err = rrf.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("poll of config from %s failed: %v", host, err)
		}
	}
	s, err := rrf.FullStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("poll of status of %s failed: %v", host, err)
	}
	name := topicSafe(s.Name)
	return &PollResult{
		Host:              host,
		TopicFriendlyName: name,
		StateTopic:        StateTopic(cfg, name),
		Config:            cr,
		Status:            s,
	}, nil
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

func variablesFromResults(res *PollResult) []*Variable {
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

	variables := []*Variable{
		{
			field: "state",
			value: res.Status.Status.String(),
		},
		{
			field: "state_code",
			value: stateCode[res.Status.Status],
		},
		{
			field: "file_time_remaining",
			value: res.Status.TimesLeft.File,
		},
		{
			field: "filament_time_remaining",
			value: res.Status.TimesLeft.Filament,
		},
		{
			field: "layer_time_remaining",
			value: res.Status.TimesLeft.Layer,
		},
		{
			field:       "mcu_temp_min",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status.MCUTemp.Min,
		},
		{
			field:       "mcu_temp_cur",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status.MCUTemp.Cur,
		},
		{
			field:       "mcu_temp_max",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status.MCUTemp.Max,
		},
		{
			field:       "vin_min",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status.VIN.Min,
		},
		{
			field:       "vin_cur",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status.VIN.Cur,
		},
		{
			field:       "vin_max",
			units:       "V",
			deviceClass: &dcVolt,
			value:       res.Status.VIN.Max,
		},
		{
			field: "geometry",
			value: res.Status.Geometry,
		},
		{
			field: "layer",
			value: res.Status.CurrentLayer,
		},
		{
			field: "speed_requested",
			value: res.Status.Speeds.Requested,
			units: "mm/s",
		},
		{
			field: "speed_top",
			value: res.Status.Speeds.Top,
			units: "mm/s",
		},
	}
	if len(res.Status.Coordinates.XYZ) == 3 {
		for i, v := range []string{"x", "y", "z"} {
			variables = append(variables, &Variable{
				field: v,
				icon:  "mdi:axis-" + v + "-arrow",
				value: res.Status.Coordinates.XYZ[i],
			})
		}
	}
	for i := range res.Status.Coordinates.Extruder {
		variables = append(variables, &Variable{
			field: fmt.Sprintf("e%d", i),
			icon:  "mdi:mdi-printer-3d-nozzle",
			value: res.Status.Coordinates.Extruder[i],
		})
	}
	for i := range res.Status.Temps.Current {
		if res.Status.Temps.Current[i] > 1000 {
			continue
		}
		temp := fmt.Sprintf("temp%d", i)
		if len(res.Status.Temps.Names) > i && res.Status.Temps.Names[i] != "" {
			temp = res.Status.Temps.Names[i]
			if !strings.Contains(temp, "temp") {
				temp = "temp_" + temp
			}
		}
		variables = append(variables, &Variable{
			field:       temp,
			units:       "°C",
			deviceClass: &dcTemp,
			value:       res.Status.Temps.Current[i],
		})
	}
	return variables
}

func discoveryMessages(cfg *Config, res *PollResult, variables []*Variable) []*mqtt.Msg {
	availability := []ha.Availability{
		{Topic: AvailabilityTopic(cfg, "bridge")},
		{Topic: res.AvailabilityTopic},
	}
	realName := res.Status.Name

	msgs := []*mqtt.Msg{}
	for _, v := range variables {
		sensor := ha.Sensor{
			Availability:     availability,
			AvailabilityMode: "all",
			Name:             realName + " " + v.field,
			Icon:             "mdi:printer-3d",
			UniqueID:         res.TopicFriendlyName + "_" + v.field,
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
		msgs = append(msgs, &mqtt.Msg{
			Topic:  ConfigTopic(cfg, res.TopicFriendlyName, v.field),
			Body:   sensor,
			Retain: true,
		})
	}
	return msgs
}

func resultMessage(res *PollResult, t time.Time, variables []*Variable) *mqtt.Msg {
	timestamp := float64(t.UnixNano()/1000000) / 1000
	msg := map[string]interface{}{
		"t": timestamp,
	}
	for _, v := range variables {
		msg[v.field] = v.value
	}
	return &mqtt.Msg{Topic: res.StateTopic, Body: msg, Retain: false}
}
