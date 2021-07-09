//go:generate go run github.com/99designs/gqlgen
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/debug"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/jensneuse/graphql-go-tools/examples/federation/reviews/graph"
	"github.com/jensneuse/graphql-go-tools/examples/federation/reviews/graph/generated"
)

const defaultPort = "4003"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", GraphQLEndpointHandler())

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func GraphQLEndpointHandler() http.Handler {
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	srv.Use(&debug.Tracer{})

	return srv
}
