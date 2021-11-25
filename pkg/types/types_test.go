package types

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Status_String(t *testing.T) {
	assert.Equal(t, "configuring", Configuring.String())
	assert.Equal(t, "idle", Idle.String())
	assert.Equal(t, "busy", Busy.String())
	assert.Equal(t, "printing", Printing.String())
	assert.Equal(t, "pausing", Pausing.String())
	assert.Equal(t, "stopped", Stopped.String())
	assert.Equal(t, "resuming", Resuming.String())
	assert.Equal(t, "halted", Halted.String())
	assert.Equal(t, "flashing", Flashing.String())
	assert.Equal(t, "toolchanging", ToolChanging.String())
	assert.Equal(t, "unknown", Status("?").String())
}

func Test_TempState_String(t *testing.T) {
	assert.Equal(t, "off", Off.String())
	assert.Equal(t, "standby", Standby.String())
	assert.Equal(t, "active", Active.String())
	assert.Equal(t, "fault", Fault.String())
	assert.Equal(t, "unknown", TempState(-1).String())
}

func Test_ScannerStatus_String(t *testing.T) {
	assert.Equal(t, "disconnected", ScannerDisconnected.String())
	assert.Equal(t, "idle", ScannerIdle.String())
	assert.Equal(t, "scanning", ScannerScanning.String())
	assert.Equal(t, "uploading", ScannerUploading.String())
	assert.Equal(t, "unknown", ScannerStatus("?").String())
}

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
			assert.Equal(t, tc.want, fanRPMs)
		})
	}
}

func Test_StatusResponse1(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponse
		wantErr bool
	}{
		{
			name: "Status 1 idle response",
			file: "testdata/type-1-idle-status.json",
			want: StatusResponse{
				Status: Idle,
				Coordinates: StatusCoords{
					AxesHomed:       []RRFBool{false, false, false},
					Extruder:        []float64{0},
					WorkplaceSystem: 1,
					XYZ:             []float64{0, 0, 550.008},
					Machine:         []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent:      []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					SpeedFactor:     100,
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
					Current: []float64{
						2000, 22.3, 2000, 2000,
						2000, 2000, 2000, 2000,
					},
					State: []TempState{
						Off, Active, Off, Off,
						Off, Off, Off, Off,
					},
				},
				UpTime: 567,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status StatusResponse
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, status)
		})
	}
}

func Test_StatusResponse2(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponse
		wantErr bool
	}{
		{
			name: "Status 2 idle response",
			file: "testdata/type-2-idle-status.json",
			want: StatusResponse{
				Status: Idle,
				Coordinates: StatusCoords{
					AxesHomed:       []RRFBool{false, false, false},
					Extruder:        []float64{0},
					WorkplaceSystem: 1,
					XYZ:             []float64{0, 0, 550.008},
					Machine:         []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent: []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					FanNames: []string{
						"", "", "", "", "", "", "", "", "",
					},
					SpeedFactor:     100,
					ExtruderFactors: []float64{100},
				},
				Sensors: Sensors{
					FanRPM: FanRPMs{0},
				},
				Temps: Temps{
					Current: []float64{
						2000, 22.3, 2000, 2000,
						2000, 2000, 2000, 2000,
					},
					State: []TempState{
						Off, Active, Off, Off,
						Off, Off, Off, Off,
					},
					Names: []string{
						"", "", "", "",
						"", "", "", "",
					},
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
				Compensation:           "None",
				ControllableFans:       2,
				TempLimit:              290,
				Endstops:               4080,
				FirmwareName:           "RepRapFirmware for Duet 2 WiFi/Ethernet",
				Geometry:               "delta",
				Axes:                   3,
				TotalAxes:              3,
				AxisNames:              "XYZ",
				Volumes:                2,
				MountedVolumes:         1,
				Name:                   "Cerb",
				Tools: []Tool{
					{
						Number:  0,
						Heaters: []int{1},
						Drives:  []int{0},
						AxisMap: [][]int{{0}, {1}},
						Fans:    1,
						Offsets: []float64{0, 0, 0},
					},
				},
				MCUTemp: &MinCurMax{
					Min: 31,
					Cur: 38.4,
					Max: 38.6,
				},
				VIN: &MinCurMax{
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
			var status StatusResponse
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, status)
		})
	}
}

func Test_StatusResponse3(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    StatusResponse
		wantErr bool
	}{
		{
			name: "Status 3 idle response",
			file: "testdata/type-3-idle-status.json",
			want: StatusResponse{
				Status: Idle,
				Coordinates: StatusCoords{
					AxesHomed:       []RRFBool{false, false, false},
					Extruder:        []float64{0},
					WorkplaceSystem: 1,
					XYZ:             []float64{0, 0, 550.008},
					Machine:         []float64{0, 0, 550.008},
				},
				Params: Params{
					FanPercent:      []float64{0, 50, 0, 0, 0, 0, 0, 0, 0},
					SpeedFactor:     100,
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
					Current: []float64{
						2000, 22.3, 2000, 2000,
						2000, 2000, 2000, 2000,
					},
					State: []TempState{
						Off, Active, Off, Off,
						Off, Off, Off, Off,
					},
				},
				UpTime:  567,
				ExtrRaw: []float64{0},
			},
		},
		{
			name: "Status 3 response while printing",
			file: "testdata/response-three-printing.json",
			want: StatusResponse{
				Status: Printing,
				Coordinates: StatusCoords{
					AxesHomed:       []RRFBool{true, true, true},
					Extruder:        []float64{461.8},
					WorkplaceSystem: 1,
					XYZ:             []float64{151.008, 23.354, 2.7},
					Machine:         []float64{53.864, 31.160, 2.400},
				},
				Speeds: Speeds{
					Requested: 50,
					Top:       50,
				},
				CurrentTool: 0,
				Params: Params{
					ATXPower:        false,
					FanPercent:      []float64{30, 0},
					SpeedFactor:     100,
					ExtruderFactors: []float64{100},
					BabyStep:        0.0,
				},
				Seq: 2,
				Sensors: Sensors{
					ProbeValue: 0.0,
					FanRPM:     []float64{-1.0, -1.0},
				},
				Temps: Temps{
					Current: []float64{
						72, 219.7,
					},
					State: []TempState{
						Active, Active,
					},
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
				UpTime:             4885.0,
				CurrentLayer:       10,
				CurrentLayerTime:   0.7,
				ExtrRaw:            []float64{464.8},
				FractionPrinted:    47.3,
				FilePosition:       290795,
				FirstLayerDuration: 207.5,
				FirstLayerHeight:   0.25,
				PrintDuration:      1583.2,
				WarmUpDuration:     541.3,
				TimesLeft: TimesLeft{
					File:     728.8,
					Filament: 722.5,
					Layer:    551.3,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			assert.NoError(t, err)
			var status StatusResponse
			err = json.Unmarshal(data, &status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, status)
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
			assert.Equal(t, tc.want, status)
		})
	}
}

func Test_RRFBool_Unmarshal(t *testing.T) {
	var rb RRFBool
	err := json.Unmarshal([]byte("1"), &rb)
	assert.NoError(t, err)
	assert.True(t, bool(rb))
	err = json.Unmarshal([]byte("0"), &rb)
	assert.NoError(t, err)
	assert.False(t, bool(rb))
}

func Test_RRFBool_Marshal(t *testing.T) {
	b, err := json.Marshal(RRFBool(true))
	assert.NoError(t, err)
	assert.Equal(t, []byte("1"), b)
	b, err = json.Marshal(RRFBool(false))
	assert.NoError(t, err)
	assert.Equal(t, []byte("0"), b)
}
