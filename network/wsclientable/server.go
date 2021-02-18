package wsclientable

import (
	"crypto/tls"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

//Idea:
//  Over an http route (preferably ssl), clients Connect and upgrade to websocket.
//  	Example: www.yyy.com/http_route?userName=test&password=123456bestPW
//      Based on the parameters the implementation will authenticate the client.
//         If authentication fails the connection is not upgraded and silently(for the client), fails (minimal work)
//  From then on clients can send json messages over the websocket connection.
//      Base-Format: {"type":"<mType>", "data":"<arbitrary implementation specific data>"}
//      The concrete server implementation will handle those messages according to the type.

type MessageHandlers map[string]func(mType string, client ClientConnection, message map[string]interface{})
type ConnOpenedHandlers []func(ClientConnection)
type ConnClosedHandlers []func(connectionID string, closeCode int, closeReason string)
type ServerClosedHandlers []func()

type Server struct {
	raw *http.Server

	authenticate         func(url.Values) (string, error)
	connOpenedHandlers   ConnOpenedHandlers
	connClosedHandlers   ConnClosedHandlers
	serverClosedHandlers ServerClosedHandlers
	// any here registered message handlers will be called upon a message of specified type
	//   remaining json map will contain the parsed data field
	//   the first argument will be the type again, in case we use the same func
	//   not thread safe - expected that message handlers are added only before server starts
	messageHandlers MessageHandlers
}

func NewWSHandlingServer() Server {
	return Server{
		authenticate: func(url.Values) (string, error) {
			return "", AuthenticationError{Reason: "No authenticator set."}
		},
		connOpenedHandlers:   ConnOpenedHandlers{},
		connClosedHandlers:   ConnClosedHandlers{},
		serverClosedHandlers: ServerClosedHandlers{},
		messageHandlers:      make(MessageHandlers),
	}
}

func (s *Server) AddMessageHandlers(messageHandlers MessageHandlers) {
	for key, element := range messageHandlers {
		s.AddMessageHandler(key, element)
	}
}
func (s *Server) AddMessageHandler(mType string, handler func(string, ClientConnection, map[string]interface{})) {
	if s.messageHandlers[mType] != nil {
		panic("Attempted to add duplicate message handler for message type")
	}
	s.messageHandlers[mType] = handler
}
func (s *Server) AddConnOpenedHandler(handler func(ClientConnection)) {
	s.connOpenedHandlers = append(s.connOpenedHandlers, handler)
}

func (s *Server) AddConnClosedHandler(handler func(connectionID string, closeCode int, closeReason string)) {
	s.connClosedHandlers = append(s.connClosedHandlers, handler)
}
func (s *Server) AddServerClosedHandler(handler func()) {
	s.serverClosedHandlers = append(s.serverClosedHandlers, handler)
}
func (s *Server) SetAuthenticator(authenticator func(url.Values) (string, error)) {
	s.authenticate = authenticator
}
func (s *Server) Close() error {
	for _, handler := range s.serverClosedHandlers {
		handler()
	}
	return s.raw.Close()
}

// never returns without error - also when locally closed.
//   Connection will only be http. Some clients(browsers) have opted to disallow unencrypted http connections.
// additionalRoutes will be added to server handler by handler.HandleFunc (must not contain conflicting patterns)
func (s *Server) StartUnencrypted(bindAddress string, bindPort int, httpWsUpgradeRoute string, additionalRoutes ...HttpRouteFunc) error {
	handler := http.NewServeMux()
	for _, routeFunc := range additionalRoutes {
		handler.HandleFunc(routeFunc.pattern, routeFunc.handler)
	}
	handler.HandleFunc(httpWsUpgradeRoute, s.upgradeAndHandleNewClient)
	server := http.Server{
		Addr:      bindAddress + ":" + strconv.Itoa(bindPort),
		Handler:   handler,
		TLSConfig: nil,
	}
	s.raw = &server
	return server.ListenAndServe()
}

type HttpRouteFunc struct {
	pattern string
	handler func(http.ResponseWriter, *http.Request)
}

func NewHttpRouteFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) HttpRouteFunc {
	return HttpRouteFunc{
		pattern: pattern,
		handler: handler,
	}
}

// never returns without error - also when locally closed.
//   For the certificate to be accepted by the client they must be from a client-local-trusted ca.
func (s *Server) StartWithTLS(bindAddress string, bindPort int, httpRoute string, tlsConfig CertAndKeyPaths) error {
	handler := http.NewServeMux()
	handler.HandleFunc(httpRoute, s.upgradeAndHandleNewClient)

	server := http.Server{ //nolint:exhaustivestruct
		Addr:      bindAddress + ":" + strconv.Itoa(bindPort),
		Handler:   handler,
		TLSConfig: nil,
	}
	s.raw = &server
	return server.ListenAndServeTLS(tlsConfig.CertificateFilePath, tlsConfig.KeyFilePath)
}

type CertAndKeyPaths struct {
	CertificateFilePath string
	KeyFilePath         string
}

// Starts the server with the ssl certificates at the given paths.
//   For certificates to be accepted by the client they must be from a client-local-trusted ca.
func (s *Server) StartWithTLSMultipleCerts(bindAddress string, bindPort int,
	httpRoute string, tlsConfigs ...CertAndKeyPaths) error {
	if len(tlsConfigs) < 1 {
		log.Fatalf("Missing certificates. Require at least 1. " +
			"Consider using http or adding a few. Can be generated with generate_cert.go")
	}

	handler := http.NewServeMux()
	handler.HandleFunc(httpRoute, s.upgradeAndHandleNewClient)

	cfg := &tls.Config{MinVersion: tls.VersionTLS10}
	for _, tlsConfig := range tlsConfigs {
		certCont, err := ioutil.ReadFile(tlsConfig.CertificateFilePath)
		if err != nil {
			log.Fatalf("Reading cert from \"%v\", error: %v", tlsConfig.CertificateFilePath, err)
		}
		keyCont, err := ioutil.ReadFile(tlsConfig.KeyFilePath)
		if err != nil {
			log.Fatalf("Reading key from: \"%v\", error: %v", tlsConfig.KeyFilePath, err)
		}
		cert, err := tls.X509KeyPair(certCont, keyCont)
		if err != nil {
			log.Fatalf("Decoding cert, error: %v", err)
		}
		cfg.Certificates = append(cfg.Certificates, cert)
	}

	server := http.Server{ //nolint:exhaustivestruct
		Addr:      bindAddress + ":" + strconv.Itoa(bindPort),
		Handler:   handler,
		TLSConfig: cfg,
	}

	s.raw = &server

	return server.ListenAndServeTLS("", "")
}

func (s *Server) upgradeAndHandleNewClient(writer http.ResponseWriter, request *http.Request) {
	initialParams, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return
	}
	name, err := s.authenticate(initialParams)
	if err != nil {
		log.Printf("Auth Err: %v", err)
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte("Failed to authenticate - because: " + err.Error()))
		return
	}

	responseHeader := http.Header{}
	upgrader := websocket.Upgrader{ //nolint:exhaustivestruct
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(writer, request, responseHeader)
	if err != nil {
		log.Println("failed to upgrade to websocket")
		writer.WriteHeader(http.StatusUpgradeRequired)
		_, _ = writer.Write([]byte("failed to upgrade to websocket"))
		return
	}

	client := ClientConnection{ID: name, raw: conn, writeMut: new(sync.Mutex)}

	for _, connOpened := range s.connOpenedHandlers {
		connOpened(client)
	}

	closeCode, closeReason := client.ListenLoop(s.messageHandlers)

	for _, connClosed := range s.connClosedHandlers {
		connClosed(client.ID, closeCode, closeReason)
	}
}
