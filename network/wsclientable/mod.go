// Let me apologise for the mediocre package name first.
// This package adds two key functionalities to websockets:
//   1. message handling by type
//       The package adds a cute, easy way to handle messages based on the type of the message
//       Types are fully customizable
//   2. connection keeping and message relay
//       Through message handling it was simple to add message relaying
//       For that we store the incoming websocket connections.
//           (it is also out of the box possible to create virtual servers, called rooms)
//       2.1. Authentication based on initial http request params is easily customizable
package wsclientable

import (
	"fmt"
	"gopkg.in/ini.v1"
)

// error returned if there is a missing field in an http request
type MissingURLFieldError struct {
	MissingFieldName string
}

func (m MissingURLFieldError) Error() string {
	return fmt.Sprintf("missing field in url params: %v", m.MissingFieldName)
}

// Error if authentication fails on a connecting websocket
type AuthenticationError struct {
	Reason string
}

func (m AuthenticationError) Error() string {
	return "Authentication failed, because " + m.Reason
}

// Given cfg requires ssl.<child> sections.
// Each section must have a 'cert_path' and 'key_path' field.
// The resulting cert-paths can be used when starting a wsclientable-server
func ReadMultipleCertsFromCfg(cfg *ini.File) []CertAndKeyPaths {
	var list []CertAndKeyPaths
	for _, sslPair := range cfg.Section("ssl").ChildSections() {
		sslCertPath := sslPair.Key("cert_path").String()
		sslKeyPath := sslPair.Key("key_path").String()
		list = append(list, CertAndKeyPaths{CertificateFilePath: sslCertPath, KeyFilePath: sslKeyPath})
	}
	return list
}
