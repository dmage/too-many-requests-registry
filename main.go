package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/health"
	"github.com/distribution/distribution/v3/registry/handlers"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	gorhandlers "github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
)

type Quota struct {
	my sync.Mutex
	c  int
}

func (q *Quota) Set(c int) {
	q.my.Lock()
	defer q.my.Unlock()
	q.c = c
}

func (q *Quota) Get() int {
	q.my.Lock()
	defer q.my.Unlock()
	return q.c
}

func (q *Quota) AcceptRequest() bool {
	q.my.Lock()
	defer q.my.Unlock()
	if q.c > 0 {
		q.c--
		return true
	}
	return q.c < 0
}

func quotaHandler(handler http.Handler) http.Handler {
	q := &Quota{
		c: -1,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if q.AcceptRequest() {
				handler.ServeHTTP(w, r)
			} else {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(http.StatusTooManyRequests)
			}
			return
		}

		var c int
		problem := ""
		if r.Method == http.MethodPost {
			var err error
			r.ParseForm()
			raw := r.PostForm.Get("c")
			c, err = strconv.Atoi(raw)
			if err == nil {
				q.Set(c)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			problem = err.Error()
		}
		c = q.Get()
		if c < 0 {
			fmt.Fprintf(w, "<h1>Quota: %d (unlimited)</h1>", c)
		} else {
			fmt.Fprintf(w, "<h1>Quota: %d</h1>", c)
		}
		if problem != "" {
			fmt.Fprintf(w, "<h2>Problem: %s</h2>", problem)
		}
		fmt.Fprintf(w, `<form method="post" action="/"><input name="c" value="" placeholder="new value"><input type="submit"></form>`)
	})
}

func alive(path string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func panicHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logrus.Panic(fmt.Sprintf("%v", err))
			}
		}()
		handler.ServeHTTP(w, r)
	})
}

func main() {
	ctx := context.Background()

	config := &configuration.Configuration{
		Storage: configuration.Storage{
			"inmemory": configuration.Parameters{},
		},
	}

	app := handlers.NewApp(ctx, config)
	app.RegisterHealthChecks()
	handler := alive("/", app)
	handler = health.Handler(handler)
	handler = quotaHandler(handler)
	handler = panicHandler(handler)
	handler = gorhandlers.CombinedLoggingHandler(os.Stdout, handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
	logrus.Fatal(server.ListenAndServe())
}
