package protocol

// Version holds the consumer target version string for the Lava protocol.
type Version struct {
	ConsumerTarget string `json:"consumer_target"`
}

// DefaultVersion is the version embedded in binary releases.
var DefaultVersion = Version{
	ConsumerTarget: "6.1.0",
}
