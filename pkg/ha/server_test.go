package ha

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beanz/rrf-go/pkg/mock"
	_ "github.com/beanz/rrf-go/pkg/types"
)

func Test_PollDevice(t *testing.T) {
	var buf bytes.Buffer
	m := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(m.Router())
	defer ts.Close()

	host := strings.Split(ts.URL, "://")[1]
	fmt.Fprintf(os.Stderr, "Using mock %s\n", host)
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
		Status2:           mock.StatusResponse(2, 0),
		Status3:           mock.StatusResponse(3, 1),
	}, r)
}
