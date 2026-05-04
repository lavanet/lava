package metrics

import "os"

// defaultHostname returns the OS hostname; used as the OTel
// service.instance.id when the operator hasn't supplied one explicitly.
func defaultHostname() (string, error) {
	return os.Hostname()
}
