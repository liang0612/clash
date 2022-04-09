package route

import (
	"encoding/json"
	R "github.com/Dreamacro/clash/rule"
	"io/ioutil"
	"net/http"

	"github.com/Dreamacro/clash/tunnel"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type addRuleRequest struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Target  string `json:"target"`
}

func ruleRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getRules)
	r.Post("/", addRules)
	return r
}

func addRules(writer http.ResponseWriter, request *http.Request) {
	req := addRuleRequest{}
	body, _ := ioutil.ReadAll(request.Body)
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}
	parsed, parseErr := R.ParseRule(req.Type, req.Payload, req.Target, nil)
	if parseErr != nil {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}
	rawRules := tunnel.Rules()
	rawRules = append(rawRules, parsed)
	tunnel.UpdateRules(rawRules)
	render.JSON(writer, request, render.M{
		"data": true,
	})
}

type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

func getRules(w http.ResponseWriter, r *http.Request) {
	rawRules := tunnel.Rules()

	rules := []Rule{}
	for _, rule := range rawRules {
		rules = append(rules, Rule{
			Type:    rule.RuleType().String(),
			Payload: rule.Payload(),
			Proxy:   rule.Adapter(),
		})
	}

	render.JSON(w, r, render.M{
		"rules": rules,
	})
}
