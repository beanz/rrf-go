package netrrf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/beanz/rrf-go/pkg/types"
	"github.com/stretchr/testify/assert"
)

type httpClientMock struct {
	responses []*http.Response
	errors    []error
	requests  []*http.Request
}

func (c *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req)
	if len(c.responses) == 0 {
		return &http.Response{}, fmt.Errorf("empty mock")
	}
	resp := c.responses[0]
	c.responses = c.responses[1:]
	var err error
	if len(c.errors) != 0 {
		err = c.errors[0]
		c.errors = c.errors[1:]
	}
	return resp, err
}

func Test_Authenticate(t *testing.T) {
	tests := []struct {
		name      string
		responses []*http.Response
		errors    []error
		checks    func(*testing.T, []*http.Request)
		wantErr   bool
	}{
		{
			name: "successful authentication",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 0, \"sessionTimeout\": 8000, \"boardType\": \"duetwifi10\"}")),
				},
			},
			checks: func(t *testing.T, r []*http.Request) {
				assert.Equal(t, 1, len(r))
				assert.Equal(t, "GET", r[0].Method)
				assert.Equal(t, "/rr_connect", r[0].URL.Path)
				assert.Equal(t, "password=foo", r[0].URL.RawQuery)
			},
			wantErr: false,
		},
		{
			name: "unsuccessful authentication",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 1, \"sessionTimeout\": 8000, \"boardType\": \"duetwifi10\"}")),
				},
			},
			wantErr: true,
		},
		{
			name: "truncated authentication response",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 1,")),
				},
			},
			wantErr: true,
		},
		{
			name:    "non-200 HTTP response",
			errors:  []error{fmt.Errorf("mock error")},
			wantErr: true,
		},
		{
			name: "error reading authentication response",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(iotest.ErrReader(fmt.Errorf("mock error"))),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var httpClient = &httpClientMock{
				tc.responses, tc.errors, []*http.Request{}}
			rrf := NewClient("localhost", "foo")
			rrf.WithHTTPClient(httpClient)
			err := rrf.Authenticate(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.checks(t, httpClient.requests)
		})
	}
}

func Test_AuthenticationError(t *testing.T) {
	err := AuthenticationError{ErrorCode: 1}
	assert.Equal(t, "authentication failed with error code=1", err.Error())
}

func Test_WithTimeout(t *testing.T) {
	rrf := NewClient("localhost", "foo")
	rrf.WithTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, rrf.timeout)
}

func Test_Config(t *testing.T) {
	tests := []struct {
		name      string
		responses []*http.Response
		errors    []error
		checks    func(*testing.T, *types.ConfigResponse)
		authDone  bool
		wantErr   bool
	}{
		{
			name: "successful config request",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 0, \"sessionTimeout\": 8000, \"boardType\": \"duetwifi10\"}")),
				},
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"axisMins\":[-100,-100,-0.2],\"axisMaxes\":[100,100,550.01],\"accelerations\":[3000,3000,3000,1000,1000,250,250,250,250,250,250,250],\"currents\":[800,800,800,500,500,0,0,0,0,0,0,0],\"firmwareElectronics\":\"Duet WiFi 1.0 or 1.01\",\"firmwareName\":\"RepRapFirmware for Duet 2 WiFi/Ethernet\",\"firmwareVersion\":\"2.05.1\",\"dwsVersion\":\"1.23\",\"firmwareDate\":\"2020-02-09b1\",\"sysdir\":\"0:/sys/\",\"idleCurrentFactor\":60,\"idleTimeout\":30,\"minFeedrates\":[20,20,20,10,10,2,2,2,2,2,2,2],\"maxFeedrates\":[333.33,333.33,333.33,60,60,20,20,20,20,20,20,20]}")),
				},
			},
			checks: func(t *testing.T, cfg *types.ConfigResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name:     "successful config request (pre-auth)",
			authDone: true,
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"axisMins\":[-100,-100,-0.2],\"axisMaxes\":[100,100,550.01],\"accelerations\":[3000,3000,3000,1000,1000,250,250,250,250,250,250,250],\"currents\":[800,800,800,500,500,0,0,0,0,0,0,0],\"firmwareElectronics\":\"Duet WiFi 1.0 or 1.01\",\"firmwareName\":\"RepRapFirmware for Duet 2 WiFi/Ethernet\",\"firmwareVersion\":\"2.05.1\",\"dwsVersion\":\"1.23\",\"firmwareDate\":\"2020-02-09b1\",\"sysdir\":\"0:/sys/\",\"idleCurrentFactor\":60,\"idleTimeout\":30,\"minFeedrates\":[20,20,20,10,10,2,2,2,2,2,2,2],\"maxFeedrates\":[333.33,333.33,333.33,60,60,20,20,20,20,20,20,20]}")),
				},
			},
			checks: func(t *testing.T, cfg *types.ConfigResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name: "truncated config response",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 0, \"sessionTimeout\": 8000, \"boardType\": \"duetwifi10\"}")),
				},
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
		{
			name: "truncated config auth response",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var httpClient = &httpClientMock{
				tc.responses, tc.errors, []*http.Request{}}
			rrf := NewClient("localhost", "foo").WithHTTPClient(httpClient)
			rrf.authDone = tc.authDone
			cfg, err := rrf.Config(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.checks(t, cfg)
		})
	}
}

func Test_RequestError(t *testing.T) {
	// invalid host to cause request creation error
	rrf := NewClient("\n", "foo")
	var resp int
	err := rrf.Request(context.Background(), "/", &resp)
	assert.Error(t, err)
}

func Test_Status1(t *testing.T) {
	tests := []struct {
		name      string
		responses []*http.Response
		errors    []error
		checks    func(*testing.T, *types.StatusResponse)
		authDone  bool
		wantErr   bool
	}{
		{
			name: "successful status1 request",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"err\": 0, \"sessionTimeout\": 8000, \"boardType\": \"duetwifi10\"}")),
				},
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						`{"status":"I","coords":{"axesHomed":[0,0,0],"wpl":1,"xyz":[0,0,550.008],"machine":[0,0,550.008],"extr":[0]},"speeds":{"requested":0,"top":0},"currentTool":0,"params":{"atxPower":0,"fanPercent":[0,50,0,0,0,0,0,0,0],"speedFactor":100,"extrFactors":[100],"babystep":0},"seq":0,"sensors":{"probeValue":0,"fanRPM":0},"temps":{"current":[2000,22.3,2000,2000,2000,2000,2000,2000],"state":[0,2,0,0,0,0,0,0],"tools":{"active":[[0]],"standby":[[0]]},"extra":[{"name":"*MCU","temp":38.3}]},"time":567}`)),
				},
			},
			checks: func(t *testing.T, s1 *types.StatusResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name:     "successful status1 request (pre-auth)",
			authDone: true,
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						"{\"axisMins\":[-100,-100,-0.2],\"axisMaxes\":[100,100,550.01],\"accelerations\":[3000,3000,3000,1000,1000,250,250,250,250,250,250,250],\"currents\":[800,800,800,500,500,0,0,0,0,0,0,0],\"firmwareElectronics\":\"Duet WiFi 1.0 or 1.01\",\"firmwareName\":\"RepRapFirmware for Duet 2 WiFi/Ethernet\",\"firmwareVersion\":\"2.05.1\",\"dwsVersion\":\"1.23\",\"firmwareDate\":\"2020-02-09b1\",\"sysdir\":\"0:/sys/\",\"idleCurrentFactor\":60,\"idleTimeout\":30,\"minFeedrates\":[20,20,20,10,10,2,2,2,2,2,2,2],\"maxFeedrates\":[333.33,333.33,333.33,60,60,20,20,20,20,20,20,20]}")),
				},
			},
			checks: func(t *testing.T, s1 *types.StatusResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name: "status1 request auth error",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
		{
			name:     "unsuccessful status1 request",
			authDone: true,
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var httpClient = &httpClientMock{
				tc.responses, tc.errors, []*http.Request{}}
			rrf := NewClient("localhost", "foo").WithHTTPClient(httpClient)
			rrf.authDone = tc.authDone
			s1, err := rrf.Status1(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.checks(t, s1)
		})
	}
}

func Test_Status2(t *testing.T) {
	tests := []struct {
		name      string
		responses []*http.Response
		errors    []error
		checks    func(*testing.T, *types.StatusResponse)
		authDone  bool
		wantErr   bool
	}{
		{
			name:     "successful status2 request",
			authDone: true,
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						`{"status":"I","coords":{"axesHomed":[0,0,0],"wpl":1,"xyz":[0,0,550.008],"machine":[0,0,550.008],"extr":[0]},"speeds":{"requested":0,"top":0},"currentTool":0,"params":{"atxPower":0,"fanPercent":[0,50,0,0,0,0,0,0,0],"fanNames":["","","","","","","","",""],"speedFactor":100,"extrFactors":[100],"babystep":0},"seq":0,"sensors":{"probeValue":0,"fanRPM":0},"temps":{"current":[2000,22.3,2000,2000,2000,2000,2000,2000],"state":[0,2,0,0,0,0,0,0],"names":["","","","","","","",""],"tools":{"active":[[0]],"standby":[[0]]},"extra":[{"name":"*MCU","temp":38.4}]},"time":567,"coldExtrudeTemp":160,"coldRetractTemp":90,"compensation":"None","controllableFans":2,"tempLimit":290,"endstops":4080,"firmwareName":"RepRapFirmware for Duet 2 WiFi/Ethernet","geometry":"delta","axes":3,"totalAxes":3,"axisNames":"XYZ","volumes":2,"mountedVolumes":1,"name":"Cerb","probe":{"threshold":500,"height":-0.2,"type":4},"tools":[{"number":0,"heaters":[1],"drives":[0],"axisMap":[[0],[1]],"fans":1,"filament":"","offsets":[0,0,0]}],"mcutemp":{"min":31,"cur":38.4,"max":38.6},"vin":{"min":11.9,"cur":12.1,"max":12.2}}`)),
				},
			},
			checks: func(t *testing.T, s1 *types.StatusResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name: "status2 request auth error",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var httpClient = &httpClientMock{
				tc.responses, tc.errors, []*http.Request{}}
			rrf := NewClient("localhost", "foo").WithHTTPClient(httpClient)
			rrf.authDone = tc.authDone
			s2, err := rrf.Status2(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.checks(t, s2)
		})
	}
}

func Test_Status3(t *testing.T) {
	tests := []struct {
		name      string
		responses []*http.Response
		errors    []error
		checks    func(*testing.T, *types.StatusResponse)
		authDone  bool
		wantErr   bool
	}{
		{
			name:     "successful status3 request",
			authDone: true,
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body: io.NopCloser(strings.NewReader(
						`{"status":"I","coords":{"axesHomed":[0,0,0],"wpl":1,"xyz":[0,0,550.008],"machine":[0,0,550.008],"extr":[0]},"speeds":{"requested":0,"top":0},"currentTool":0,"params":{"atxPower":0,"fanPercent":[0,50,0,0,0,0,0,0,0],"speedFactor":100,"extrFactors":[100],"babystep":0},"seq":0,"sensors":{"probeValue":0,"fanRPM":0},"temps":{"current":[2000,22.3,2000,2000,2000,2000,2000,2000],"state":[0,2,0,0,0,0,0,0],"tools":{"active":[[0]],"standby":[[0]]},"extra":[{"name":"*MCU","temp":38.4}]},"time":567,"currentLayer":0,"currentLayerTime":0,"extrRaw":[0],"fractionPrinted":0,"filePosition":0,"firstLayerDuration":0,"firstLayerHeight":0,"printDuration":0,"warmUpDuration":0,"timesLeft":{"file":0,"filament":0,"layer":0}}`)),
				},
			},
			checks: func(t *testing.T, s1 *types.StatusResponse) {
				assert.Equal(t, 1, 1)
			},
			wantErr: false,
		},
		{
			name: "status3 request auth error",
			responses: []*http.Response{
				{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.0",
					Body:       io.NopCloser(strings.NewReader("{")),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var httpClient = &httpClientMock{
				tc.responses, tc.errors, []*http.Request{}}
			rrf := NewClient("localhost", "foo").WithHTTPClient(httpClient)
			rrf.authDone = tc.authDone
			s3, err := rrf.Status3(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.checks(t, s3)
		})
	}
}
