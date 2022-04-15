package route

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Dreamacro/clash/adapter/provider"
	rules2 "github.com/Dreamacro/clash/rule"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dreamacro/clash/adapter"
	"github.com/Dreamacro/clash/adapter/outboundgroup"
	"github.com/Dreamacro/clash/component/profile/cachefile"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/tunnel"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func proxyRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getProxies)
	r.Post("/", addProxy)
	r.Get("/stop/{name}", stopProxy)
	r.Route("/{name}", func(r chi.Router) {
		r.Use(parseProxyName, findProxyByName)
		r.Get("/", getProxy)
		r.Get("/delay", getProxyDelay)
		r.Put("/", updateProxy)
	})
	return r
}

func stopProxy(writer http.ResponseWriter, request *http.Request) {
	name := getEscapeParam(request, "name")
	name = strings.Trim(name, " ")
	if len(name) == 0 {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}
	proxies := tunnel.Proxies()
	providers := tunnel.Providers()
	delete(proxies, name)
	delete(providers, name)

	rules := tunnel.Rules()
	var newRules []C.Rule
	for _, item := range rules {
		if adapter := item.Adapter(); adapter == name {
			dis := item.(*rules2.ListenerPort)
			if dis != nil {
				dis.Dispose()
			}
			continue
		}
		newRules = append(newRules, item)
	}

	tunnel.UpdateProxies(proxies, providers)
	tunnel.UpdateRules(newRules)
}

func addProxy(writer http.ResponseWriter, request *http.Request) {
	var req map[string]interface{}
	body, _ := ioutil.ReadAll(request.Body)
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}

	proxy, err := adapter.ParseProxy(req)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.JSON(writer, request, ErrBadRequest)
		return
	}
	fmt.Println(proxy)
	groupname := proxy.Name()
	ps := []C.Proxy{}
	ps = append(ps, proxy)
	hc := provider.NewHealthCheck(ps, "", 0, true)
	pd, _ := provider.NewCompatibleProvider(groupname, ps, hc)
	providers := tunnel.Providers()
	providers[groupname] = pd

	proxies := tunnel.Proxies()
	proxies[groupname] = proxy
	tunnel.UpdateProxies(proxies, providers)
	render.JSON(writer, request, render.M{
		"data": true,
	})
}

func parseProxyName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := getEscapeParam(r, "name")
		ctx := context.WithValue(r.Context(), CtxKeyProxyName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func findProxyByName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Context().Value(CtxKeyProxyName).(string)
		proxies := tunnel.Proxies()
		proxy, exist := proxies[name]
		if !exist {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyProxy, proxy)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getProxies(w http.ResponseWriter, r *http.Request) {
	proxies := tunnel.Proxies()
	render.JSON(w, r, render.M{
		"proxies": proxies,
	})
}

func getProxy(w http.ResponseWriter, r *http.Request) {
	proxy := r.Context().Value(CtxKeyProxy).(C.Proxy)
	render.JSON(w, r, proxy)
}

type UpdateProxyRequest struct {
	Name string `json:"name"`
}

func updateProxy(w http.ResponseWriter, r *http.Request) {
	req := UpdateProxyRequest{}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}

	proxy := r.Context().Value(CtxKeyProxy).(*adapter.Proxy)
	selector, ok := proxy.ProxyAdapter.(*outboundgroup.Selector)
	if !ok {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, newError("Must be a Selector"))
		return
	}

	if err := selector.Set(req.Name); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, newError(fmt.Sprintf("Selector update error: %s", err.Error())))
		return
	}

	cachefile.Cache().SetSelected(proxy.Name(), req.Name)
	render.NoContent(w, r)
}

func getProxyDelay(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	url := query.Get("url")
	timeout, err := strconv.ParseInt(query.Get("timeout"), 10, 16)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}

	proxy := r.Context().Value(CtxKeyProxy).(C.Proxy)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	delay, err := proxy.URLTest(ctx, url)
	if ctx.Err() != nil {
		render.Status(r, http.StatusGatewayTimeout)
		render.JSON(w, r, ErrRequestTimeout)
		return
	}

	if err != nil || delay == 0 {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, newError("An error occurred in the delay test"))
		return
	}

	render.JSON(w, r, render.M{
		"delay": delay,
	})
}
