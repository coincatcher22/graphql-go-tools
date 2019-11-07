package http

import (
	"bytes"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (g *GraphQLHTTPRequestHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		g.log.Error("GraphQLHTTPRequestHandler.handleHTTP",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	executor, rootNode, ctx, err := g.executionHandler.Handle(data)
	if err != nil {
		g.log.Error("executionHandler.Handle",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	_, err = executor.Execute(ctx, rootNode, buf)
	if err != nil {
		g.log.Error("executor.Execute",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	_, _ = buf.WriteTo(w)
}
