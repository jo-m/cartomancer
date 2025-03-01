package endpoints

import (
	"database/sql"
	"fmt"
	"goweb/internal/pkg/endpoints/users"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type router struct {
	db *sql.DB
}

func (rt router) hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello")
}

func New(db *sql.DB) http.Handler {
	rt := &router{db: db}

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Timeout(60 * time.Second))

	mux.Get("/hello", rt.hello)

	users := users.New(db)
	mux.Mount("/users", users)

	return mux
}
