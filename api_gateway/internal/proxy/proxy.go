// internal/proxy/proxy.go в api_gateway
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

// singleJoiningSlash (без изменений)
func singleJoiningSlash(a, b string) string {
	// ... (код как был) ...
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		if b == "" {
			return a
		}
		return a + "/" + b
	}
	return a + b
}

// --- ИЗМЕНЕНИЕ: NewReverseProxy теперь принимает prefixToStrip ---
// NewReverseProxy создает настроенный обратный прокси для Gin
func NewReverseProxy(target, prefixToStrip string, log *logrus.Logger) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Errorf("Failed to parse target URL '%s': %v", target, err)
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director // Сохраняем оригинальный director
	proxy.Director = func(req *http.Request) {
		// Сначала выполняем стандартный director
		originalDirector(req)

		// Сохраняем оригинальный путь для логирования и возможного использования
		originalPath := req.URL.Path
		log.Debugf("Proxy Director: Original path from Gin: %s", originalPath)

		// --- ИЗМЕНЕНИЕ: Удаляем префикс из пути ---
		if prefixToStrip != "" && strings.HasPrefix(req.URL.Path, prefixToStrip) {
			// Используем RawPath, если он есть, иначе Path
			// Это важно для путей, содержащих закодированные символы вроде %2F
			hasRawPath := req.URL.RawPath != ""
			originalRawOrPath := req.URL.Path
			if hasRawPath {
				originalRawOrPath = req.URL.RawPath
			}

			// Удаляем префикс
			newPath := strings.TrimPrefix(originalRawOrPath, prefixToStrip)

			// Гарантируем, что путь начинается с / или пустой
			if newPath == "" {
				newPath = "/" // Если остался пустой путь, делаем его корневым
			} else if !strings.HasPrefix(newPath, "/") {
				newPath = "/" + newPath // Добавляем слеш, если его нет
			}

			// Обновляем Path и RawPath
			req.URL.Path = newPath
			if hasRawPath {
				// Пытаемся обновить RawPath соответственно (может быть сложно, если были кодированные символы в префиксе)
				// Простой вариант - просто установить его равным Path, если префикс не содержал спецсимволов
				req.URL.RawPath = newPath // Может потребоваться более сложная логика URL-кодирования здесь
			}

			log.Debugf("Proxy Director: Stripped prefix '%s'. New path: %s (RawPath: %s)", prefixToStrip, req.URL.Path, req.URL.RawPath)
		} else {
			log.Debugf("Proxy Director: Prefix '%s' not found or empty. Path remains: %s", prefixToStrip, req.URL.Path)
		}
		// --- КОНЕЦ ИЗМЕНЕНИЯ ---

		// Переписываем Host
		req.Host = targetURL.Host

		// Удаляем заголовок Authorization
		req.Header.Del("Authorization")

		// Добавляем X-User-ID
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

		// Логируем финальный URL
		log.Infof("Proxy Director: Final request URL being sent: %s", req.URL.String())
	}

	// Обработчик ошибок проксирования (без изменений)
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		// ... (как было) ...
		log.Errorf("Reverse proxy error to target '%s' for path '%s': %v", target, req.URL.Path, err)
		http.Error(rw, "Bad Gateway", http.StatusBadGateway)
	}

	log.Infof("Reverse proxy created for target: %s (will strip prefix: '%s')", target, prefixToStrip)
	return proxy, nil
}

// ProxyHandler (без изменений)
// Он только передает userID в контекст, Director делает основную работу
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

		originalPath := c.Request.URL.Path // Логируем путь до вызова ServeHTTP
		log.Debugf("ProxyHandler: Forwarding request for path '%s' (UserID: %d)", originalPath, userID)

		p.ServeHTTP(c.Writer, c.Request)
	}
}
