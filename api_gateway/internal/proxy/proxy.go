package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings" // <-- Добавлен импорт

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewReverseProxy(target, prefixToStrip string, log *logrus.Logger) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Errorf("Failed to parse target URL '%s': %v", target, err)
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director // Сохраняем оригинальный director
	proxy.Director = func(req *http.Request) {

		originalDirector(req)

		originalPath := req.URL.Path
		log.Debugf("Proxy Director: Original path from Gin: %s", originalPath)

		if prefixToStrip != "" && strings.HasPrefix(req.URL.Path, prefixToStrip) {

			hasRawPath := req.URL.RawPath != ""
			originalRawOrPath := req.URL.Path
			if hasRawPath {
				originalRawOrPath = req.URL.RawPath
			}

			newPath := strings.TrimPrefix(originalRawOrPath, prefixToStrip)

			if newPath == "" {
				newPath = "/"
			} else if !strings.HasPrefix(newPath, "/") {
				newPath = "/" + newPath
			}

			req.URL.Path = newPath
			if hasRawPath {

				req.URL.RawPath = newPath
			}

			log.Debugf("Proxy Director: Stripped prefix '%s'. New path: %s (RawPath: %s)", prefixToStrip, req.URL.Path, req.URL.RawPath)
		} else {
			log.Debugf("Proxy Director: Prefix '%s' not found or empty. Path remains: %s", prefixToStrip, req.URL.Path)
		}

		req.Host = targetURL.Host

		req.Header.Del("Authorization")

		if userIDVal := req.Context().Value("ginUserID"); userIDVal != nil {
			if userID, ok := userIDVal.(int); ok && userID > 0 {
				req.Header.Set("X-User-ID", strconv.Itoa(userID))
				log.Debugf("Proxying request with X-User-ID: %d to %s", userID, targetURL.String())
			} else {
				log.Warnf("Found ginUserID in context but it's not a valid int: %v", userIDVal)
			}
		} else {
			log.Debugf("Proxying request without X-User-ID to %s", targetURL.String())
		}

		log.Infof("Proxy Director: Final request URL being sent: %s", req.URL.String())
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {

		log.Errorf("Reverse proxy error to target '%s' for path '%s': %v", target, req.URL.Path, err)
		http.Error(rw, "Bad Gateway", http.StatusBadGateway)
	}

	log.Infof("Reverse proxy created for target: %s (will strip prefix: '%s')", target, prefixToStrip)
	return proxy, nil
}

func ProxyHandler(p *httputil.ReverseProxy, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("userID")
		var userID int = 0
		if exists {
			if id, ok := userIDVal.(int); ok {
				userID = id
			} else {
				log.Warnf("userID found in context but is not int: %v", userIDVal)
			}
		}

		ctx := context.WithValue(c.Request.Context(), "ginUserID", userID)
		c.Request = c.Request.WithContext(ctx)

		originalPath := c.Request.URL.Path
		log.Debugf("ProxyHandler: Forwarding request for path '%s' (UserID: %d)", originalPath, userID)

		p.ServeHTTP(c.Writer, c.Request)
		log.Infof(">>> ProxyHandler: About to call ServeHTTP for %s", c.Request.URL.Path) // <-- ДОБАВЬ ЭТОТ ЛОГ
		p.ServeHTTP(c.Writer, c.Request)
		log.Infof("<<< ProxyHandler: ServeHTTP finished for %s", c.Request.URL.Path) // <-- И ЭТОТ ЛОГ
	}
}
