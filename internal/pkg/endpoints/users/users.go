package users

//go:generate go tool qtc -dir=.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/endpoints/users/tpl"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const urlParamName = "name"

type Users struct {
	db *sql.DB
}

// http://127.0.0.1:8050/users
func (s *Users) getUsers(w http.ResponseWriter, r *http.Request) {
	u, err := db.New(s.db).GetUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(u)
}

// http://127.0.0.1:8050/users/test
func (s *Users) getUser(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, urlParamName)

	u, err := db.New(s.db).GetUserByName(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := tpl.MainPage{
		Username: u.Username,
	}
	tpl.WritePageTemplate(w, &p)
}

func New(db *sql.DB) http.Handler {
	svc := &Users{db: db}

	mux := chi.NewRouter()
	mux.Get("/users", svc.getUsers)
	mux.Get(fmt.Sprintf("/users/{%s}", urlParamName), svc.getUser)

	return mux
}
