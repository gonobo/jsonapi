package null

import "net/http"

// Writer does nothing. It implements [http.ResponseWriter].
type Writer struct{}

func (Writer) Header() http.Header       { return http.Header{} }
func (Writer) Write([]byte) (int, error) { return 0, nil }
func (Writer) WriteHeader(int)           {}

// Handler does nothing. It implements [http.Handler].
type Handler struct{}

func (Handler) ServeHTTP(http.ResponseWriter, *http.Request) {}
