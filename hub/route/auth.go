package route

import (
	"encoding/json"
	"github.com/Dreamacro/clash/component/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"io/ioutil"
	"net/http"
)

type addAuthRequest struct {
	Port int    `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

func authRouter() http.Handler {
	r := chi.NewRouter()
	r.Post("/", addAuth)
	return r
}

func addAuth(writer http.ResponseWriter, request *http.Request) {
	req := addAuthRequest{}
	body, _ := ioutil.ReadAll(request.Body)
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}
	auth.AddAuth(req.Port, req.User, req.Pass)
	render.JSON(writer, request, render.M{
		"data": true,
	})
}
