package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// gzipResponseWriter wraps ResponseWriter to intercept writes
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// gzipWriterPool reuses gzip.Writers across requests. A fresh writer allocates
// ~200KB+ of compressor state each time, which is real GC pressure under load;
// pooling reuses that state via Reset.
//
// NOTE: KrakenD OSS cannot compress responses, so this server-side gzip is the
// only compressor in the path — keep it, just pool it. Make sure KrakenD proxies
// these routes in passthrough/no-op encoding so the compressed body reaches the
// client intact instead of being decoded at the gateway.
var gzipWriterPool = sync.Pool{
	New: func() any {
		gz, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return gz
	},
}

// GzipMiddleware compresses responses for clients that support it
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			gz.Close()
			gzipWriterPool.Put(gz)
		}()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // length is no longer accurate
		next.ServeHTTP(gzipResponseWriter{gz, w}, r)
	})
}
