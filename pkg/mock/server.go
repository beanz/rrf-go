package mock

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/beanz/rrf-go/pkg/types"

	"github.com/go-chi/chi"
)

type MockRRF struct {
	Auth     *types.AuthResponse
	logger   *log.Logger
	auth     bool
	count    float64
	requests int
	failSet  map[int]bool
	reply    string
	d        float64
	mu       sync.Mutex
}

const toRad = float64(0.0174533)
const d = float64(100)

func NewMockRRF(log *log.Logger) *MockRRF {
	m := &MockRRF{
		Auth: &types.AuthResponse{
			ErrorCode:      0,
			SessionTimeout: types.Time(8000),
			BoardType:      "mockrrf",
		},
		logger:   log,
		auth:     false,
		count:    0,
		requests: 0,
		failSet:  map[int]bool{},
		d:        d,
	}
	return m
}

func (m *MockRRF) WithFailSet(f map[int]bool) *MockRRF {
	m.failSet = f
	return m
}

func round(f float64) float64 {
	return math.Round(f*1000) / 1000
}

func (m *MockRRF) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
}

func (m *MockRRF) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.logger.Print("Request: ", r.URL)
			rn := m.requests
			m.requests++
			if m.failSet[rn] {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	router.Get("/rr_connect", m.connectHandler())
	router.Get("/rr_config", m.configHandler())
	router.Get("/rr_status", m.statusHandler())
	router.Get("/rr_reply", m.replyHandler())
	router.Get("/rr_gcode", m.gcodeHandler())
	router.Get("/rr_filelist", m.filelistHandler())
	router.Get("/rr_fileinfo", m.fileinfoHandler())
	router.Get("/rr_download", m.downloadHandler())
	root := "./static"
	fs := http.FileServer(http.Dir(root))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		r.RequestURI = r.URL.Path
		if r.RequestURI == "/" {
			r.RequestURI = "/index.html"
			r.URL.Path = "/index.html"
		}
		if _, err := os.Stat(root + r.RequestURI); !os.IsNotExist(err) {
			m.logger.Print("Serving file")
			fs.ServeHTTP(w, r)
		} else {
			if _, err := os.Stat(root + r.RequestURI + ".gz"); !os.IsNotExist(err) {
				m.logger.Print("Serving .gz file")
				w.Header().Set("Content-Encoding", "gzip")
				ext := filepath.Ext(r.RequestURI)
				switch ext {
				case ".html":
					m.logger.Print("setting html content type")
					w.Header().Set("Content-Type", "text/html")
				case ".js":
					m.logger.Print("setting js content type")
					w.Header().Set("Content-Type", "application/javascript")
				case ".css":
					m.logger.Print("setting css content type")
					w.Header().Set("Content-Type", "text/css")
				}
				r.RequestURI = r.RequestURI + ".gz"
				r.URL.Path = r.URL.Path + ".gz"
				fs.ServeHTTP(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}
	})
	return router
}

func (m *MockRRF) connectHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pw := r.URL.Query().Get("password")
		w.Header().Set("Content-Type", "application/json")
		ar := &types.AuthResponse{ErrorCode: 1}
		if pw == "passw0rd" || pw == "reprap" {
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
		err := json.NewEncoder(w).Encode(ConfigResponse())
		if err != nil {
			m.logger.Printf("failed to encode config response: %v\n", err)
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
		w.Header().Set("Content-Type", "application/json")
		kind, err := strconv.Atoi(r.URL.Query().Get("type"))
		if err != nil || (kind != 1 && kind != 2 && kind != 3) {
			kind = 1
		}
		resp := StatusResponse(kind, m.count)
		m.Update()
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", resp, err)
		}
	}
}

func (m *MockRRF) replyHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte(m.reply))
		if err != nil {
			m.logger.Printf("failed to write reply %s: %v\n", m.reply, err)
		}
	}
}

func (m *MockRRF) gcodeHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		//gcode := r.URL.Query().Get("gcode")
		resp := map[string]interface{}{
			"buff": 250,
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", resp, err)
		}
	}
}

func (m *MockRRF) filelistHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		dir := r.URL.Query().Get("dir")
		resp := map[string]interface{}{
			"dir":   dir,
			"first": 0,
			"files": []string{},
			"next":  0,
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", resp, err)
		}
	}
}

func (m *MockRRF) fileinfoHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"err": 1,
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			m.logger.Printf("failed to encode %v: %v\n", resp, err)
		}
	}
}

func (m *MockRRF) downloadHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.auth {
			m.logger.Printf("no authorised for %v\n", r)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func ConfigResponse() *types.ConfigResponse {
	return &types.ConfigResponse{
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
	}
}

func StatusResponse(kind int, count float64) *types.StatusResponse {
	s := &types.StatusResponse{
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
			Current: []float64{80, 200, 2000, 2000},
			State: []types.TempState{
				types.Active, types.Active, types.Off, types.Off,
			},
			Names: []string{"bed", "", "", ""},
			Tools: types.ToolTemps{
				Active:  [][]float64{{0}},
				Standby: [][]float64{{0}},
			},
			Extra: []types.ExtraTemps{},
		},
		UpTime: 500,
	}

	switch kind {
	case 2:
		s.ColdExtrudeTemperature = 160
		s.ColdRetractTemperature = 90
		s.Compensation = "None"
		s.ControllableFans = 2
		s.TempLimit = 290
		s.Endstops = 4080
		s.FirmwareName = "RepRapFirmware for Duet 2 WiFi/Ethernet"
		s.Geometry = "delta"
		s.Axes = 3
		s.TotalAxes = 3
		s.AxisNames = "XYZ"
		s.Volumes = 2
		s.MountedVolumes = 1
		s.Params.FanNames = []string{"", "print"}
		s.Name = "MockRRF"
		s.Probe = types.Probe{
			Threshold: 500,
			Height:    -0.2,
			Type:      4,
		}
		s.Tools = []types.Tool{
			{
				Number:  0,
				Heaters: []int{1},
				Drives:  []int{0},
				AxisMap: [][]int{{0}, {1}},
				Fans:    1,
				Offsets: []float64{0, 0, 0},
			},
		}
		s.MCUTemp = &types.MinCurMax{Min: 31, Cur: 38.4, Max: 38.6}
		s.VIN = &types.MinCurMax{Min: 11.9, Cur: 12.1, Max: 12.2}
	case 3:
		s.CurrentLayerTime = 20
		s.ExtrRaw = []float64{0}
		s.FirstLayerDuration = 10
		s.FirstLayerHeight = 0.2
		s.WarmUpDuration = 2
	}

	s.UpTime = types.Time(count)

	layer := count
	if layer > 100 {
		s.Status = types.Idle
		return s
	}

	rad := count * toRad
	sin := math.Sin(rad)
	cos := math.Cos(rad)
	xyz := []float64{
		round(d * cos),
		round(d * sin),
		round(d + d*sin),
	}

	s.Temps.Current = []float64{
		round(80 + 5*sin), round(200 + 5*cos), 2000, 2000}
	s.Coordinates.XYZ = xyz
	s.Coordinates.Machine = xyz

	if kind != 3 {
		return s
	}

	tl := types.Time((100 - count) * 20)

	s.PrintDuration = types.Time(count)
	s.TimesLeft = types.TimesLeft{
		File:     tl,
		Filament: tl,
		Layer:    tl,
	}
	s.CurrentLayer = int(count)
	s.FractionPrinted = count
	s.FilePosition = 20 * int(count)
	return s
}

func FullStatusResponse(count float64) *types.StatusResponse {
	res := StatusResponse(2, count)
	s3 := StatusResponse(3, count+1)

	res.CurrentLayer = s3.CurrentLayer
	res.CurrentLayerTime = s3.CurrentLayerTime
	res.ExtrRaw = s3.ExtrRaw
	res.FractionPrinted = s3.FractionPrinted
	res.FilePosition = s3.FilePosition
	res.FirstLayerDuration = s3.FirstLayerDuration
	res.FirstLayerHeight = s3.FirstLayerHeight
	res.PrintDuration = s3.PrintDuration
	res.WarmUpDuration = s3.WarmUpDuration
	res.TimesLeft = s3.TimesLeft

	return res
}
