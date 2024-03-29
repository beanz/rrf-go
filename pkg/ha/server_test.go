package ha

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"strings"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mqtt "github.com/beanz/homeassistant-go/pkg/mqtt"
	ha "github.com/beanz/homeassistant-go/pkg/types"
	"github.com/beanz/rrf-go/pkg/mock"
	"github.com/beanz/rrf-go/pkg/types"
)

func Test_PollDevice(t *testing.T) {
	var buf bytes.Buffer
	m := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(m.Router())
	defer ts.Close()

	host := strings.Split(ts.URL, "://")[1]
	ctx, cancel := context.WithCancel(context.Background())
	r, err := pollDevice(ctx,
		host, &Config{
			Password:             "passw0rd",
			Interval:             60,
			TopicPrefix:          "rrfdata",
			DiscoveryTopicPrefix: "rrfdisc",
		}, true)
	defer cancel()

	require.NoError(t, err)
	assert.Equal(t, &PollResult{
		Host:              host,
		TopicFriendlyName: "mockrrf",
		StateTopic:        "rrfdata/mockrrf/state",
		Config:            mock.ConfigResponse(),
		Status:            mock.FullStatusResponse(0),
	}, r)
}

func Test_PollDeviceErrorCases(t *testing.T) {
	for _, call := range []int{0, 1, 2, 3} {
		t.Run(fmt.Sprintf("fail on call #%d", call), func(t *testing.T) {

			var buf bytes.Buffer
			m := mock.NewMockRRF(log.New(&buf, "", 0)).WithFailSet(
				map[int]bool{call: true},
			)

			ts := httptest.NewServer(m.Router())
			defer ts.Close()

			host := strings.Split(ts.URL, "://")[1]
			ctx, cancel := context.WithCancel(context.Background())
			_, err := pollDevice(ctx,
				host, &Config{
					Password:             "passw0rd",
					Interval:             60,
					TopicPrefix:          "rrfdata",
					DiscoveryTopicPrefix: "rrfdisc",
				}, true)
			defer cancel()

			require.Error(t, err)
		})
	}
}

func Test_VariablesFromResults(t *testing.T) {
	v := variablesFromResults(&PollResult{
		Host:              "foo",
		TopicFriendlyName: "mockrrf",
		AvailabilityTopic: "rrfdata/mockrrf/availability",
		StateTopic:        "rrfdata/mockrrf/state",
		Config:            mock.ConfigResponse(),
		Status:            mock.FullStatusResponse(0),
	})
	dcTemp := ha.DeviceClassTemperature
	dcVolt := ha.DeviceClassVoltage
	assert.Equal(t, []*Variable{
		{
			field: "state",
			value: "printing",
		},
		{
			field: "state_code",
			value: 3,
		},
		{
			field: "file_time_remaining",
			value: types.Time(1980),
		},
		{
			field: "filament_time_remaining",
			value: types.Time(1980),
		},
		{
			field: "layer_time_remaining",
			value: types.Time(1980),
		},
		{
			field:       "mcu_temp_min",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       31.0,
		},
		{
			field:       "mcu_temp_cur",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       38.4,
		},
		{
			field:       "mcu_temp_max",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       38.6,
		},
		{
			field:       "vin_min",
			units:       "V",
			deviceClass: &dcVolt,
			value:       11.9,
		},
		{
			field:       "vin_cur",
			units:       "V",
			deviceClass: &dcVolt,
			value:       12.1,
		},
		{
			field:       "vin_max",
			units:       "V",
			deviceClass: &dcVolt,
			value:       12.2,
		},
		{
			field: "geometry",
			value: "delta",
		},
		{
			field: "layer",
			value: 1,
		},
		{
			field: "speed_requested",
			value: 20.0,
			units: "mm/s",
		},
		{
			field: "speed_top",
			value: 30.0,
			units: "mm/s",
		},
		{
			field: "x",
			icon:  "mdi:axis-x-arrow",
			value: 100.0,
		},
		{
			field: "y",
			icon:  "mdi:axis-y-arrow",
			value: 0.0,
		},
		{
			field: "z",
			icon:  "mdi:axis-z-arrow",
			value: 100.0,
		},
		{
			field: "e0",
			icon:  "mdi:mdi-printer-3d-nozzle",
			value: 0.0,
		},
		{
			field:       "temp_bed",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       80.0,
		},
		{
			field:       "temp1",
			units:       "°C",
			deviceClass: &dcTemp,
			value:       205.0,
		},
	}, v)
}

func Test_DiscoveryMessages(t *testing.T) {
	dcTemp := ha.DeviceClassTemperature
	tests := []struct {
		name     string
		variable *Variable
		want     *mqtt.Msg
	}{
		{
			name:     "simple state",
			variable: &Variable{field: "state", value: "printing"},
			want: &mqtt.Msg{
				Topic: "rrfdisc/sensor/mockrrf_state/config",
				Body: ha.Sensor{
					Availability: []ha.Availability{
						{
							Topic: "rrfdata/bridge/availability",
						},
						{
							Topic: "rrfdata/mockrrf/availability",
						},
					},
					AvailabilityMode: "all",
					Device: ha.Device{
						ConfigurationURL: "http://foo",
						Identifiers:      []string{"mockrrf", "mockrrf_state"},
						Model:            "Duet WiFi 1.0 or 1.01",
						Name:             "MockRRF",
						SwVersion:        "RepRapFirmware for Duet 2 WiFi/Ethernet v2.05.1 (2020-02-09b1)",
					},
					Icon:          "mdi:printer-3d",
					Name:          "MockRRF state",
					StateTopic:    "rrfdata/mockrrf/state",
					UniqueID:      "mockrrf_state",
					ValueTemplate: "{{ value_json.state}}",
				},
				Retain: true,
			},
		},
		{
			name: "variable with icon, device class and units",
			variable: &Variable{
				field:       "hotend_temp",
				icon:        "mdi:mdi-printer-3d-nozzle",
				units:       "°C",
				deviceClass: &dcTemp,
				value:       200.3,
			},
			want: &mqtt.Msg{
				Topic: "rrfdisc/sensor/mockrrf_hotend_temp/config",
				Body: ha.Sensor{
					Availability: []ha.Availability{
						{
							Topic: "rrfdata/bridge/availability",
						},
						{
							Topic: "rrfdata/mockrrf/availability",
						},
					},
					AvailabilityMode: "all",
					Device: ha.Device{
						ConfigurationURL: "http://foo",
						Identifiers: []string{"mockrrf",
							"mockrrf_hotend_temp"},
						Model:     "Duet WiFi 1.0 or 1.01",
						Name:      "MockRRF",
						SwVersion: "RepRapFirmware for Duet 2 WiFi/Ethernet v2.05.1 (2020-02-09b1)",
					},
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Icon:              "mdi:mdi-printer-3d-nozzle",
					Name:              "MockRRF hotend_temp",
					StateTopic:        "rrfdata/mockrrf/state",
					UniqueID:          "mockrrf_hotend_temp",
					ValueTemplate:     "{{ value_json.hotend_temp}}",
				},
				Retain: true,
			},
		},
	}
	res := &PollResult{
		Host:              "foo",
		TopicFriendlyName: "mockrrf",
		AvailabilityTopic: "rrfdata/mockrrf/availability",
		StateTopic:        "rrfdata/mockrrf/state",
		Config:            mock.ConfigResponse(),
		Status:            mock.FullStatusResponse(0),
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs := discoveryMessages(&Config{
				DiscoveryTopicPrefix: "rrfdisc",
				TopicPrefix:          "rrfdata",
			}, res, []*Variable{tc.variable})
			assert.Equal(t, tc.want, msgs[0])
		})
	}
}

func Test_ResultMessages(t *testing.T) {
	then, err := time.Parse("2006-01-02", "2021-11-19")
	assert.NoError(t, err)
	msg := resultMessage(
		&PollResult{StateTopic: "rrfdata/mockrrf/state"},
		then,
		[]*Variable{{field: "state", value: "printing"}},
	)
	assert.Equal(t, &mqtt.Msg{
		Topic: "rrfdata/mockrrf/state",
		Body: map[string]interface{}{
			"state": "printing",
			"t":     1637280000.0,
		},
	}, msg)
}

func Test_DeviceLoop(t *testing.T) {
	var buf bytes.Buffer
	mock := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(mock.Router())
	defer ts.Close()

	host := strings.Split(ts.URL, "://")[1]
	safeHost := topicSafe(host)

	msgc := make(chan *mqtt.Msg, 100)
	var pollBuf bytes.Buffer
	polllog := log.New(&pollBuf, "", 0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		deviceLoop(ctx,
			host, &Config{
				Password:             "passw0rd",
				Interval:             time.Second * 60,
				TopicPrefix:          "rrfdata",
				DiscoveryTopicPrefix: "rrfdisc",
			}, msgc, polllog)
	}()
	defer cancel()

	msg := <-msgc
	assert.Equal(t,
		&mqtt.Msg{
			Topic:  "rrfdata/" + safeHost + "/availability",
			Body:   "online",
			Retain: true,
		},
		msg)
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	count := 0
LOOP:
	for {
		select {
		case <-msgc:
			count++
		case <-timeout.C:
			break LOOP
		}
	}
	assert.Equal(t, 22, count) // 21 discovery messages + state
}

func Test_DeviceLoopError(t *testing.T) {
	var buf bytes.Buffer
	mock := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(mock.Router())
	defer ts.Close()

	host := strings.Split(ts.URL, "://")[1]
	safeHost := topicSafe(host)

	msgc := make(chan *mqtt.Msg, 100)
	var pollBuf bytes.Buffer
	polllog := log.New(&pollBuf, "", 0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		deviceLoop(ctx,
			host, &Config{
				Password:             "incorrect",
				Interval:             time.Second * 60,
				TopicPrefix:          "rrfdata",
				DiscoveryTopicPrefix: "rrfdisc",
			}, msgc, polllog)
	}()
	defer cancel()

	msg := <-msgc
	assert.Equal(t,
		&mqtt.Msg{
			Topic:  "rrfdata/" + safeHost + "/availability",
			Body:   "offline",
			Retain: true,
		},
		msg)
}

func Test_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var buf bytes.Buffer
	mock := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(mock.Router())
	defer ts.Close()
	host := strings.Split(ts.URL, "://")[1]

	cfg := &Config{
		Password:             "passw0rd",
		Interval:             time.Second * 60,
		TopicPrefix:          "rrfdata",
		DiscoveryTopicPrefix: "rrfdisc",
		Devices:              []string{host},
	}
	msgp := make(chan *mqtt.Msg, 30)
	msgs := make(chan *mqtt.Msg, 1)
	var runBuf bytes.Buffer
	log := log.New(&runBuf, "", 0)
	go func() {
		err := Run(ctx, cfg, log, &MockPS{}, msgp, msgs)
		assert.NoError(t, err)
	}()

	safeHost := topicSafe(host)
	msg := <-msgp
	assert.Equal(t,
		&mqtt.Msg{
			Topic:  "rrfdata/" + safeHost + "/availability",
			Body:   "online",
			Retain: true,
		},
		msg)

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	count := 0
LOOP:
	for {
		select {
		case <-msgp:
			count++
		case <-timeout.C:
			break LOOP
		}
	}
	assert.Equal(t, 22, count) // 21 discovery messages + state
}

type MockPS struct {
}

func (ps *MockPS) Run(ctx context.Context, msgp, msgs chan *mqtt.Msg) error {

	<-ctx.Done()
	return nil
}
