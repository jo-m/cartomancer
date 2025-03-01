package svc

import (
	"context"
	"fmt"
	"net/http"
)

const sessionCookieName = "session"

type SessionCookieWriter struct {
	value string
}

func (c *SessionCookieWriter) VisitPostApiV1LoginResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    c.value,
		Quoted:   true,
		MaxAge:   1800,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	fmt.Fprint(w, "Set cookie")

	return nil
}

/*
curl -X POST --data '{"username":"test","password":"asdf"}' -H 'content-type: application/json' http://127.0.0.1:8050/api/v1/login
*/
func (s *Server) PostApiV1Login(ctx context.Context, request PostApiV1LoginRequestObject) (PostApiV1LoginResponseObject, error) {
	// TODO: password check.
	fmt.Println(request.Body)
	return &SessionCookieWriter{value: "asdf"}, nil
}
