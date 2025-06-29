// Package playback contains the playback server.
package playback

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/FlyInkTV/FlyInkStream-Engine/internal/auth"
	"github.com/FlyInkTV/FlyInkStream-Engine/internal/conf"
	"github.com/FlyInkTV/FlyInkStream-Engine/internal/logger"
	"github.com/FlyInkTV/FlyInkStream-Engine/internal/protocols/httpp"
	"github.com/FlyInkTV/FlyInkStream-Engine/internal/restrictnetwork"
	"github.com/gin-gonic/gin"
)

type serverAuthManager interface {
	Authenticate(req *auth.Request) error
}

// Server is the playback server.
type Server struct {
	Address        string
	Encryption     bool
	ServerKey      string
	ServerCert     string
	AllowOrigin    string
	TrustedProxies conf.IPNetworks
	ReadTimeout    conf.Duration
	PathConfs      map[string]*conf.Path
	AuthManager    serverAuthManager
	Parent         logger.Writer

	httpServer *httpp.Server
	mutex      sync.RWMutex
}

// Initialize initializes Server.
func (s *Server) Initialize() error {
	router := gin.New()
	router.SetTrustedProxies(s.TrustedProxies.ToTrustedProxies()) //nolint:errcheck

	router.Use(s.middlewareOrigin)

	router.GET("/list", s.onList)
	router.GET("/get", s.onGet)

	network, address := restrictnetwork.Restrict("tcp", s.Address)

	s.httpServer = &httpp.Server{
		Network:     network,
		Address:     address,
		ReadTimeout: time.Duration(s.ReadTimeout),
		Encryption:  s.Encryption,
		ServerCert:  s.ServerCert,
		ServerKey:   s.ServerKey,
		Handler:     router,
		Parent:      s,
	}
	err := s.httpServer.Initialize()
	if err != nil {
		return err
	}

	s.Log(logger.Info, "listener opened on "+address)

	return nil
}

// Close closes Server.
func (s *Server) Close() {
	s.Log(logger.Info, "listener is closing")
	s.httpServer.Close()
}

// Log implements logger.Writer.
func (s *Server) Log(level logger.Level, format string, args ...interface{}) {
	s.Parent.Log(level, "[playback] "+format, args...)
}

// ReloadPathConfs is called by core.Core.
func (s *Server) ReloadPathConfs(pathConfs map[string]*conf.Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.PathConfs = pathConfs
}

func (s *Server) writeError(ctx *gin.Context, status int, err error) {
	// show error in logs
	s.Log(logger.Error, err.Error())

	// add error to response
	ctx.String(status, err.Error())
}

func (s *Server) safeFindPathConf(name string) (*conf.Path, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	pathConf, _, err := conf.FindPathConf(s.PathConfs, name)
	return pathConf, err
}

func (s *Server) middlewareOrigin(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", s.AllowOrigin)
	ctx.Header("Access-Control-Allow-Credentials", "true")

	// preflight requests
	if ctx.Request.Method == http.MethodOptions &&
		ctx.Request.Header.Get("Access-Control-Request-Method") != "" {
		ctx.Header("Access-Control-Allow-Methods", "OPTIONS, GET")
		ctx.Header("Access-Control-Allow-Headers", "Authorization")
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}
}

func (s *Server) doAuth(ctx *gin.Context, pathName string) bool {
	req := &auth.Request{
		Action:      conf.AuthActionPlayback,
		Path:        pathName,
		Query:       ctx.Request.URL.RawQuery,
		Credentials: httpp.Credentials(ctx.Request),
		IP:          net.ParseIP(ctx.ClientIP()),
	}

	err := s.AuthManager.Authenticate(req)
	if err != nil {
		if err.(auth.Error).AskCredentials { //nolint:errorlint
			ctx.Header("WWW-Authenticate", `Basic realm="FlyInkStream-Engine"`)
			ctx.Writer.WriteHeader(http.StatusUnauthorized)
			return false
		}

		s.Log(logger.Info, "connection %v failed to authenticate: %v",
			httpp.RemoteAddr(ctx), err.(*auth.Error).Message) //nolint:errorlint

		// wait some seconds to mitigate brute force attacks
		<-time.After(auth.PauseAfterError)

		ctx.Writer.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}




