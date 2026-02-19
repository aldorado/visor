package observability

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
)

func RecoverMiddleware(component string, next http.Handler) http.Handler {
	log := Component(component)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := compactStack(string(debug.Stack()))
				log.Error(r.Context(), "panic recovered",
					"panic", fmt.Sprintf("%v", rec),
					"method", r.Method,
					"path", r.URL.Path,
					"traceback", stack,
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func compactStack(stack string) string {
	lines := strings.Split(stack, "\n")
	if len(lines) <= 16 {
		return strings.TrimSpace(stack)
	}
	return strings.TrimSpace(strings.Join(lines[:16], "\n"))
}
