/*
Copyright (c) 2021 Mark Hindess

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
package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type AuthResponse struct {
	ErrorCode      int    `json:"err"`
	SessionTimeout Time   `json:"sessionTimeout,omitempty"`
	BoardType      string `json:"boardType,omitempty"`
}

type StatusResponse struct {
	Status      Status       `json:"status,omitempty"`
	Coordinates StatusCoords `json:"coords,omitempty"`
	Speeds      Speeds       `json:"speeds,omitempty"`
	CurrentTool int          `json:"currentTool,omitempty"`
	Output      *Output      `json:"output,omitempty"`
	Params      Params       `json:"params,omitempty"`
	Seq         int          `json:"seq,omitempty"`
	Sensors     Sensors      `json:"sensors,omitempty"`
	Temps       Temps        `json:"temps,omitempty"`
	Resp        string       `json:"resp,omitempty"`
	UpTime      Time         `json:"time,omitempty"`
	Scanner     *Scanner     `json:"scanner,omitempty"`
	Spindles    []Spindle    `json:"spindles,omitempty"`

	// type 2 properties
	ColdExtrudeTemperature float64          `json:"coldExtrudeTemp,omitempty"`
	ColdRetractTemperature float64          `json:"coldRetractTemp,omitempty"`
	Compensation           Compensation     `json:"compensation,omitempty"`
	ControllableFans       ControllableFans `json:"controllableFans,omitempty"`
	TempLimit              float64          `json:"tempLimit,omitempty"`
	Endstops               EndstopState     `json:"endstops,omitempty"`
	FirmwareName           string           `json:"firmwareName,omitempty"`
	Geometry               string           `json:"geometry,omitempty"`
	Axes                   int              `json:"axes,omitempty"`
	TotalAxes              int              `json:"totalAxes,omitempty"`
	AxisNames              string           `json:"axisNames,omitempty"`
	Volumes                int              `json:"volumes,omitempty"`
	MountedVolumes         VolumeState      `json:"mountedVolumes,omitempty"`
	Name                   string           `json:"name,omitempty"`
	Probe                  Probe            `json:"probe:,omitempty"`
	Tools                  []Tool           `json:"tools,omitempty"`
	MCUTemp                *MinCurMax       `json:"mcutemp,omitempty"`
	VIN                    *MinCurMax       `json:"vin,omitempty"`

	// type 3 properties
	CurrentLayer       int       `json:"currentLayer,omitempty"`
	CurrentLayerTime   Time      `json:"currentLayerTime,omitempty"`
	ExtrRaw            []float64 `json:"extrRaw,omitempty"`
	FractionPrinted    float64   `json:"fractionPrinted,omitempty"`
	FilePosition       int       `json:"filePosition,omitempty"`
	FirstLayerDuration Time      `json:"firstLayerDuration,omitempty"`
	FirstLayerHeight   float64   `json:"firstLayerHeight,omitempty"`
	PrintDuration      Time      `json:"printDuration,omitempty"`
	WarmUpDuration     Time      `json:"warmUpDuration,omitempty"`
	TimesLeft          TimesLeft `json:"timesLeft,omitempty"`
}

type ConfigResponse struct {
	AxisMins            []float64 `json:"axisMins,omitempty"`
	AxisMaxes           []float64 `json:"axisMaxes,omitempty"`
	Accelerations       []float64 `json:"accelerations,omitempty"`
	Currents            []float64 `json:"currents,omitempty"`
	FirmwareElectronics string    `json:"firmwareElectronics,omitempty"`
	FirmwareName        string    `json:"firmwareName,omitempty"`
	FirmwareVersion     string    `json:"firmwareVersion,omitempty"`
	DWSVersion          string    `json:"dwsVersion,omitempty"`
	FirmwareDate        Date      `json:"firmwareDate,omitempty"`
	SysDir              string    `json:"sysdir,omitempty"`
	IdleCurrentFactor   float64   `json:"idleCurrentFactor,omitempty"`
	IdleTimeout         float64   `json:"idleTimeout,omitempty"`
	MinFeedRates        []float64 `json:"minFeedrates,omitempty"`
	MaxFeedRates        []float64 `json:"maxFeedrates,omitempty"`
}

type RRFBool bool

func (b *RRFBool) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "1" {
		*b = true
	} else {
		*b = false
	}
	return nil
}

func (b RRFBool) MarshalJSON() ([]byte, error) {
	r := 0
	if b {
		r++
	}
	return json.Marshal(r)
}

type StatusCoords struct {
	AxesHomed       []RRFBool `json:"axesHomed,omitempty"`
	Extruder        []float64 `json:"extr,omitempty"`
	WorkplaceSystem int       `json:"wpl,omitempty"`
	XYZ             []float64 `json:"xyz,omitempty"`
	Machine         []float64 `json:"machine,omitempty"`
}

type Speeds struct {
	Requested float64 `json:"requested,omitempty"`
	Top       float64 `json:"top,omitempty"`
}

type Output struct {
	BeepDuration  int    `json:"beepDuration,omitempty"`
	BeepFrequency int    `json:"beepFrequency,omitempty"`
	Message       string `json:"message,omitempty"`
}

type Params struct {
	ATXPower        RRFBool   `json:"atxPower,omitempty"`
	FanPercent      []float64 `json:"fanPercent,omitempty"`
	FanNames        []string  `json:"fanNames,omitempty"`
	SpeedFactor     float64   `json:"speedFactor,omitempty"`
	ExtruderFactors []float64 `json:"extrFactors,omitempty"`
	BabyStep        float64   `json:"babystep,omitempty"`
}

// FanRPMs is a []float64 but the type is used to handle the special case
// described at:
// https://github.com/Duet3D/RepRapFirmware/blob/a1b2f3a7/src/RepRap.cpp#L1045
type FanRPMs []float64

func (f *FanRPMs) UnmarshalJSON(data []byte) error {
	s := string(data)
	*f = []float64{}
	if s[0] == '[' {
		s = s[1 : len(s)-1]
	}
	// strip whitespace
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	if s != "" {
		for _, vs := range strings.Split(s, ",") {
			v, err := strconv.ParseFloat(vs, 64)
			if err != nil {
				return fmt.Errorf("unmarshal fanRPM '%s': %w", vs, err)
			}
			*f = append(*f, v)
		}
	}
	return nil
}

type Sensors struct {
	ProbeValue     float64   `json:"probeValue,omitempty"`
	ProbeSecondary []float64 `json:"probeSecondary,omitempty"`
	FanRPM         FanRPMs   `json:"fanRPM,omitempty"`
}

type Status string

const (
	Configuring  Status = "C"
	Idle         Status = "I"
	Busy         Status = "B"
	Printing     Status = "P"
	Pausing      Status = "D"
	Stopped      Status = "S"
	Resuming     Status = "R"
	Halted       Status = "H"
	Flashing     Status = "F"
	ToolChanging Status = "T"
)

func (s Status) String() string {
	switch s {
	case Configuring:
		return "configuring"
	case Idle:
		return "idle"
	case Busy:
		return "busy"
	case Printing:
		return "printing"
	case Pausing:
		return "pausing"
	case Stopped:
		return "stopped"
	case Resuming:
		return "resuming"
	case Halted:
		return "halted"
	case Flashing:
		return "flashing"
	case ToolChanging:
		return "toolchanging"
	}
	return "unknown"
}

// C (configuration file is being processed)
// I (idle, no movement or code is being performed)
// B (busy, live movement is in progress or a macro file is being run)
// P (printing a file)
// D (decelerating, pausing a running print)
// S (stopped, live print has been paused)
// R (resuming a paused print)
// H (halted, after emergency stop)
// F (flashing new firmware)
// T (changing tool, new in 1.17b)

type TempState int

const (
	Off     TempState = 0
	Standby TempState = 1
	Active  TempState = 2
	Fault   TempState = 3
)

func (s TempState) String() string {
	switch s {
	case Off:
		return "off"
	case Standby:
		return "standby"
	case Active:
		return "active"
	case Fault:
		return "fault"
	default:
		return "unknown"
	}
}

type Temp struct {
	Current float64   `json:"current,omitempty"`
	Active  float64   `json:"active,omitempty"`
	Standby float64   `json:"standby,omitempty"`
	State   TempState `json:"state,omitempty"`
}

type ToolTemps struct {
	Active  [][]float64 `json:"active,omitempty"`
	Standby [][]float64 `json:"standby,omitempty"`
}

type Temps struct {
	Bed     Temp         `json:"bed,omitempty"`
	Chamber Temp         `json:"chamber,omitempty"`
	Heads   Temp         `json:"heads,omitempty"`
	Tools   ToolTemps    `json:"tools,omitempty"`
	Current []float64    `json:"current,omitempty"`
	State   []TempState  `json:"state,omitempty"`
	Names   []string     `json:"names,omitempty"`
	Extra   []ExtraTemps `json:"extra"`
}

type ExtraTemps struct {
	Name string  `json:"name,omitempty"`
	Temp float64 `json:"temp,omitempty"`
}

type Time float64 // todo parse to time.Duration

type ScannerStatus string

const (
	ScannerDisconnected ScannerStatus = "D"
	ScannerIdle         ScannerStatus = "I"
	ScannerScanning     ScannerStatus = "S"
	ScannerUploading    ScannerStatus = "U"
)

func (s ScannerStatus) String() string {
	switch s {
	case ScannerDisconnected:
		return "disconnected"
	case ScannerIdle:
		return "idle"
	case ScannerScanning:
		return "scanning"
	case ScannerUploading:
		return "uploading"
	}
	return "unknown"
}

type Scanner struct {
	Status   ScannerStatus `json:"status,omitempty"`
	Progress float64       `json:"progress,omitempty"`
}

type Spindle struct {
	Current float64 `json:"current,omitempty"`
	Active  float64 `json:"active,omitempty"`
	Tool    int     `json:"tool,omitempty"`
}

type Compensation string

type ControllableFans int // TODO: decode bitmap to []bool
type EndstopState int     // TODO: decode bitmap to []bool
type VolumeState int      // TODO: decode bitmap to []bool

type Probe struct {
	Threshold int     `json:"threshold,omitempty"`
	Height    float64 `json:"height,omitempty"`
	Type      int     `json:"type,omitempty"`
}

type Tool struct {
	Number   int       `json:"number,omitempty"`
	Name     string    `json:"name,omitempty"`
	Heaters  []int     `json:"heaters,omitempty"`
	Drives   []int     `json:"drives,omitempty"`
	AxisMap  [][]int   `json:"axisMap,omitempty"`
	Fans     int       `json:"fans,omitempty"`
	Filament string    `json:"filament,omitempty"`
	Offsets  []float64 `json:"offsets,omitempty"`
}

type MinCurMax struct {
	Min float64 `json:"min,omitempty"`
	Cur float64 `json:"cur,omitempty"`
	Max float64 `json:"max,omitempty"`
}

type TimesLeft struct {
	File     Time `json:"file,omitempty"`
	Filament Time `json:"filament,omitempty"`
	Layer    Time `json:"layer,omitempty"`
}

type Date string // TODO unmarshal to time
