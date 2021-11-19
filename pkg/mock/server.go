package mock

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sync"

	"github.com/beanz/rrf-go/pkg/types"

	"github.com/go-chi/chi"
)

type MockRRF struct {
	Config *types.ConfigResponse
	Auth   *types.AuthResponse
	S1     *types.StatusResponse
	S2     *types.StatusResponse
	S3     *types.StatusResponse
	logger *log.Logger
	auth   bool
	count  float64
	d      float64
	mu     sync.Mutex
}

const toRad = float64(0.0174533)

func NewMockRRF(log *log.Logger) *MockRRF {
	d := float64(100)
	m := &MockRRF{
		Config: &types.ConfigResponse{
			AxisMins:            []float64{-d, -d, 0},
			AxisMaxes:           []float64{d, d, 2 * d},
			Accelerations:       []float64{3000, 3000, 3000, 1000},
			Currents:            []float64{800, 800, 800, 500},
			FirmwareElectronics: "Duet WiFi 1.0 or 1.01",
			FirmwareName:        "RepRapFirmware for Duet 2 WiFi/Ethernet",
			FirmwareVersion:     "2.05.1",
			DWSVersion:          "1.23",
			FirmwareDate:        "2020-02-09b1",
			SysDir:              "0:/sys/",
			IdleCurrentFactor:   60,
			IdleTimeout:         30,
			MinFeedRates:        []float64{20, 20, 20, 10},
			MaxFeedRates:        []float64{300, 300, 300, 60},
		},
		Auth: &types.AuthResponse{
			ErrorCode:      0,
			SessionTimeout: types.Time(8000),
			BoardType:      "mockrrf",
		},
		S1: &types.StatusResponse{
			Status: types.Printing,
			Coordinates: types.StatusCoords{
				AxesHomed: []types.RRFBool{true, true, true},
				Extruder:  []float64{0},
				XYZ:       []float64{0, 0, 0},
				Machine:   []float64{0, 0, 0},
			},
			Speeds: types.Speeds{
				Requested: 20,
				Top:       30,
			},
			CurrentTool: 0,
			Params: types.Params{
				FanPercent:      []float64{0, 50},
				SpeedFactor:     100,
				ExtruderFactors: []float64{100},
			},
			Seq:     0,
			Sensors: types.Sensors{},
			Temps: types.Temps{
				Current: []float64{80, 200},
				State:   []types.TempState{types.Active, types.Active},
			},
			UpTime: 500,
		},
		logger: log,
		auth:   false,
		count:  0,
		d:      d,
	}
	m.S2 = m.S1

	m.S2.ColdExtrudeTemperature = 160
	m.S2.ColdRetractTemperature = 90
	m.S2.Compensation = "None"
	m.S2.ControllableFans = 2
	m.S2.TempLimit = 290
	m.S2.Endstops = 4080
	m.S2.FirmwareName = "RepRapFirmware for Duet 2 WiFi/Ethernet"
	m.S2.Geometry = "delta"
	m.S2.Axes = 3
	m.S2.TotalAxes = 3
	m.S2.AxisNames = "XYZ"
	m.S2.Volumes = 2
	m.S2.MountedVolumes = 1
	m.S2.Name = "MockRRF"
	m.S2.Probe = types.Probe{
		Threshold: 500,
		Height:    -0.2,
		Type:      4,
	}
	m.S2.Tools = []types.Tool{
		{
			Number:  0,
			Heaters: []int{1},
			Drives:  []int{0},
			AxisMap: [][]int{{0}, {1}},
			Fans:    1,
			Offsets: []float64{0, 0, 0},
		},
	}
	m.S2.MCUTemp = types.MinCurMax{Min: 31, Cur: 38.4, Max: 38.6}
	m.S2.VIN = types.MinCurMax{Min: 11.9, Cur: 12.1, Max: 12.2}

	m.S3 = m.S1
	m.S3.CurrentLayerTime = 20
	m.S3.ExtrRaw = []float64{0}
	m.S3.FirstLayerDuration = 10
	m.S3.FirstLayerHeight = 0.2
	m.S3.WarmUpDuration = 2

	m.Update()
	return m
}

func round(f float64) float64 {
	return math.Round(f*1000) / 1000
}

func (m *MockRRF) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.S1.UpTime = types.Time(m.count)
	m.S2.UpTime = types.Time(m.count)
	m.S3.UpTime = types.Time(m.count)

	layer := m.count
	if layer > 100 {
		m.S1.Status = types.Idle
		m.S1.Status = types.Idle
		m.S1.Status = types.Idle
		return
	}

	rad := m.count * toRad
	sin := math.Sin(rad)
	cos := math.Cos(rad)
	xyz := []float64{
		round(m.d * cos),
		round(m.d * sin),
		round(m.d + m.d*sin),
	}
	m.S1.Coordinates.XYZ = xyz
	m.S1.Coordinates.Machine = xyz
	m.S1.Temps.Current = []float64{round(80 + 5*sin), round(200 + 5*cos)}

	m.S2.Coordinates.XYZ = xyz
	m.S2.Coordinates.Machine = xyz
	m.S2.Temps.Current = m.S1.Temps.Current

	m.S3.Coordinates.XYZ = xyz
	m.S3.Coordinates.Machine = xyz
	m.S3.Temps.Current = m.S1.Temps.Current

	tl := types.Time((100 - m.count) * 20)

	m.S3.PrintDuration = types.Time(m.count)
	m.S3.TimesLeft = types.TimesLeft{
		File:     tl,
		Filament: tl,
		Layer:    tl,
	}
	m.S3.CurrentLayer = int(m.count)
	m.S3.FractionPrinted = m.count
	m.S3.FilePosition = 20 * int(m.count)

	m.count++
}

func (m *MockRRF) Router() http.Handler {
	router := chi.NewRouter()
	router.Get("/rr_connect", m.connectHandler())
	router.Get("/rr_config", m.configHandler())
	router.Get("/rr_status", m.statusHandler())
	return router
}

func (m *MockRRF) connectHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pw := r.URL.Query().Get("password")
		w.Header().Set("Content-Type", "application/json")
		ar := &types.AuthResponse{ErrorCode: 1}
		if pw == "passw0rd" {
			m.auth = true
			ar = m.Auth
		}
		err := json.NewEncoder(w).Encode(ar)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", ar, err)
		}
	}
}

func (m *MockRRF) configHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(m.Config)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", m.Config, err)
		}
	}
}

func (m *MockRRF) statusHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		m.Update()
		kind := r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		var resp interface{}
		switch kind {
		case "2":
			resp = m.S2
		case "3":
			resp = m.S3
		default: // legacy not supported so do type=1 by default
			resp = m.S1
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", resp, err)
		}
	}
}
