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

	"github.com/beanz/rrf-go/pkg/mock"
	_ "github.com/beanz/rrf-go/pkg/netrrf"
	_ "github.com/beanz/rrf-go/pkg/types"
)

func Test_PollDevice(t *testing.T) {
	var buf bytes.Buffer
	mock := mock.NewMockRRF(log.New(&buf, "", 0))
	ts := httptest.NewServer(mock.Router())
	defer ts.Close()

	host := strings.Split(ts.URL, "://")[1]
	fmt.Fprintf(os.Stderr, "Using mock %s\n", host)
	safeHost := topicSafe(host)

	msgc := make(chan *Msg, 100)
	var pollBuf bytes.Buffer
	polllog := log.New(&pollBuf, "", 0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		pollDevice(ctx,
			host, &Config{
				Password:             "passw0rd",
				Interval:             60,
				TopicPrefix:          "rrfdata",
				DiscoveryTopicPrefix: "rrfdisc",
			}, msgc, polllog)
	}()
	defer cancel()

	msg := <-msgc
	assert.Equal(t,
		&Msg{
			topic:  "rrfdata/" + safeHost + "/availability",
			body:   "online",
			retain: true,
		},
		msg)
}
