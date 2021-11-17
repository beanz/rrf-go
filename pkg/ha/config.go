package ha

import (
	"time"
)

type Config struct {
	AppName              string
	Version              string
	Devices              []string
	Password             string
	Broker               string
	ClientID             string
	TopicPrefix          string
	DiscoveryTopicPrefix string
	Interval             time.Duration
	DiscoveryInterval    time.Duration
	ConnectRetryDelay    time.Duration
	KeepAlive            int
}
