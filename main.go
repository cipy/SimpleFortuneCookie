package main

// adapted from: https://golang.cafe/blog/golang-rest-api-example.html

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
)

var (
	listFortuneRe   = regexp.MustCompile(`^/fortunes[/]*$`)
	getFortuneRe    = regexp.MustCompile(`^/fortunes[/](\d+)$`)
	createFortuneRe = regexp.MustCompile(`^/fortunes[/]*$`)
)

type fortune struct {
	ID   string `json:"id"`
	Message string `json:"message"`
}

type datastore struct {
	m map[string]fortune
	*sync.RWMutex
}

type fortuneHandler struct {
	store *datastore
}

func (h *fortuneHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method == http.MethodGet && listFortuneRe.MatchString(r.URL.Path):
		h.List(w, r)
		return
	case r.Method == http.MethodGet && getFortuneRe.MatchString(r.URL.Path):
		h.Get(w, r)
		return
	case r.Method == http.MethodPost && createFortuneRe.MatchString(r.URL.Path):
		h.Create(w, r)
		return
	default:
		notFound(w, r)
		return
	}
}

func (h *fortuneHandler) List(w http.ResponseWriter, r *http.Request) {
	h.store.RLock()
	fortunes := make([]fortune, 0, len(h.store.m))
	for _, v := range h.store.m {
		fortunes = append(fortunes, v)
	}
	h.store.RUnlock()
	jsonBytes, err := json.Marshal(fortunes)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *fortuneHandler) Get(w http.ResponseWriter, r *http.Request) {
	matches := getFortuneRe.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		notFound(w, r)
		return
	}
	h.store.RLock()
	u, ok := h.store.m[matches[1]]
	h.store.RUnlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("fortune not found"))
		return
	}
	jsonBytes, err := json.Marshal(u)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *fortuneHandler) Create(w http.ResponseWriter, r *http.Request) {
	var u fortune
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		internalServerError(w, r)
		return
	}
	h.store.Lock()
	h.store.m[u.ID] = u
	h.store.Unlock()
	jsonBytes, err := json.Marshal(u)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func internalServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("internal server error"))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func main() {
	mux := http.NewServeMux()
	fortuneH := &fortuneHandler{
		store: &datastore{
			m: map[string]fortune{
				"1": fortune{ID: "1", Message: "It ain't over till it's EOF."},
			},
			RWMutex: &sync.RWMutex{},
		},
	}
	mux.Handle("/fortunes", fortuneH)
	mux.Handle("/fortunes/", fortuneH)

	http.ListenAndServe("localhost:8080", mux)
}
