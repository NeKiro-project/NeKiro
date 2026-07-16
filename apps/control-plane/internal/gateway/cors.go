package gateway

import "net/http"

const (
	corsAllowMethods = "GET, POST, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Authorization, Content-Type, Accept"
)

type corsHandler struct {
	next           http.Handler
	allowedOrigins map[string]struct{}
}

func NewCORSHandler(next http.Handler, allowedOrigins []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return corsHandler{next: next, allowedOrigins: allowed}
}

func (handler corsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	origin := request.Header.Get("Origin")
	if _, allowed := handler.allowedOrigins[origin]; allowed {
		writer.Header().Set("Access-Control-Allow-Origin", origin)
		writer.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
		writer.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
		writer.Header().Set("Access-Control-Expose-Headers", TraceHeader)
		writer.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
		if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
	}
	handler.next.ServeHTTP(writer, request)
}
