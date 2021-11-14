package types

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FanRPM(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    FanRPMs
		wantErr bool
	}{
		{
			name:    "single float legacy response",
			data:    "300",
			want:    FanRPMs{300},
			wantErr: false,
		},
		{
			name:    "list",
			data:    "[ 300, 600 ]",
			want:    FanRPMs{300, 600},
			wantErr: false,
		},
		{
			name:    "list with newlines",
			data:    "[\n  200,\n  400\n]",
			want:    FanRPMs{200, 400},
			wantErr: false,
		},
		{
			name:    "single bad element",
			data:    "\"bar\"",
			wantErr: true,
		},
		{
			name:    "list with bad element",
			data:    "[\n  200,\n  \"foo\"\n]",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fanRPMs FanRPMs
			err := json.Unmarshal([]byte(tc.data), &fanRPMs)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, fanRPMs, tc.want)
		})
	}
}

func Test_StatusResponseOne(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponseOne
		wantErr bool
	}{
		{
			name: "Status 1 idle response",
			file: "testdata/type-1-idle-status.json",
			want: StatusResponseOne{
				Status: Idle,
				Coordinates: StatusCoords{
					AxisHomed: []RRFBool{false, false, false},
					XYZ:       []float64{0, 0, 550.008},
					Extruder:  []float64{0},
					Machine:   []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent:      []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					ExtruderFactors: []float64{100},
				},
				Sensors: Sensors{
					FanRPM: []float64{0},
				},
				Temps: Temps{
					Tools: ToolTemps{
						Active:  [][]float64{[]float64{0}},
						Standby: [][]float64{[]float64{0}},
					},
				},
				UpTime: 567,
			},
		},
		{
			name: "Status 1 response while printing",
			file: "testdata/response-one-printing.json",
			want: StatusResponseOne{
				Status: Printing,
				Coordinates: StatusCoords{
					AxisHomed: []RRFBool{true, true, true},
					XYZ:       []float64{151.008, 23.354, 2.7},
					Extruder:  []float64{461.8},
					Machine:   []float64{53.864, 31.160, 2.400},
				},
				Speeds: Speeds{
					Requested: 50,
					Top:       50,
				},
				CurrentTool: 0,
				Params: Params{
					ATXPower:   false,
					FanPercent: []float64{30, 0},
					//SpeedFactor:     []float64{100},
					ExtruderFactors: []float64{100},
					BabyStep:        0.0,
				},
				Seq: 2,
				Sensors: Sensors{
					ProbeValue: 0.0,
					FanRPM:     []float64{-1.0, -1.0},
				},
				Temps: Temps{
					Bed: Temp{
						Current: 72.0,
						Active:  72.0,
						State:   2,
					},
					Tools: ToolTemps{
						Active:  [][]float64{[]float64{220}},
						Standby: [][]float64{[]float64{220}},
					},
				},
				UpTime: 4885.0,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status StatusResponseOne
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, status, tc.want)
		})
	}
}

func Test_StatusResponseTwo(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponseTwo
		wantErr bool
	}{
		{
			name: "Status 2 idle response",
			file: "testdata/type-2-idle-status.json",
			want: StatusResponseTwo{
				Status: Idle,
				Coordinates: StatusCoords{
					AxisHomed: []RRFBool{false, false, false},
					Extruder:  []float64{0},
					XYZ:       []float64{0, 0, 550.008},
					Machine:   []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent:      []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					ExtruderFactors: []float64{100},
				},
				Sensors: Sensors{
					FanRPM: FanRPMs{0},
				},
				Temps: Temps{
					Bed: Temp{
						Current: 0,
						Active:  0,
						State:   Off,
					},
					Tools: ToolTemps{
						Active:  [][]float64{[]float64{0}},
						Standby: [][]float64{[]float64{0}},
					},
				},
				UpTime:                 567,
				ColdExtrudeTemperature: 160,
				ColdRetractTemperature: 90,
				Endstops:               4080,
				FirmwareName:           "RepRapFirmware for Duet 2 WiFi/Ethernet",
				Geometry:               "delta",
				Axes:                   3,
				Volumes:                2,
				MountedVolumes:         1,
				Name:                   "Cerb",
				Tools: []Tool{
					{
						Number:  0,
						Heaters: []int{1},
						Drives:  []int{0},
						AxisMap: [][]int{{0}, {1}},
					},
				},
				MCUTemp: MinCurMax{
					Min: 31,
					Cur: 38.4,
					Max: 38.6,
				},
				VIN: MinCurMax{
					Min: 11.9,
					Cur: 12.1,
					Max: 12.2,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status StatusResponseTwo
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, status, tc.want)
		})
	}
}

func Test_StatusResponseThree(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponseThree
		wantErr bool
	}{
		{
			name: "Status 3 idle response",
			file: "testdata/type-3-idle-status.json",
			want: StatusResponseThree{
				Status: Idle,
				Coordinates: StatusCoords{
					AxisHomed: []RRFBool{false, false, false},
					Extruder:  []float64{0},
					XYZ:       []float64{0, 0, 550.008},
					Machine:   []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent:      []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					ExtruderFactors: []float64{100},
				},
				Sensors: Sensors{
					FanRPM: FanRPMs{0},
				},
				Temps: Temps{
					Bed: Temp{
						Current: 0,
						Active:  0,
						State:   Off,
					},
					Tools: ToolTemps{
						Active:  [][]float64{[]float64{0}},
						Standby: [][]float64{[]float64{0}},
					},
				},
				UpTime:  567,
				ExtrRaw: []float64{0},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status StatusResponseThree
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, status, tc.want)
		})
	}
}

func Test_ConfigResponse(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    ConfigResponse
		wantErr bool
	}{
		{
			name: "Config response",
			file: "testdata/config-response.json",
			want: ConfigResponse{
				AxisMins:  []float64{-100, -100, -0.2},
				AxisMaxes: []float64{100, 100, 550.01},
				Accelerations: []float64{
					3000, 3000, 3000, 1000, 1000,
					250, 250, 250, 250, 250, 250, 250,
				},
				Currents: []float64{
					800, 800, 800, 500, 500,
					0, 0, 0, 0, 0, 0, 0,
				},
				FirmwareElectronics: "Duet WiFi 1.0 or 1.01",
				FirmwareName:        "RepRapFirmware for Duet 2 WiFi/Ethernet",
				FirmwareVersion:     "2.05.1",
				DWSVersion:          "1.23",
				FirmwareDate:        "2020-02-09b1",
				SysDir:              "0:/sys/",
				IdleCurrentFactor:   60,
				IdleTimeout:         30,
				MinFeedRates: []float64{
					20, 20, 20, 10, 10,
					2, 2, 2, 2, 2, 2, 2,
				},
				MaxFeedRates: []float64{
					333.33, 333.33, 333.33, 60, 60,
					20, 20, 20, 20, 20, 20, 20,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status ConfigResponse
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, status, tc.want)
		})
	}
}
