package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	hackmiddleware "github.com/jensneuse/graphql-go-tools/hack/middleware"
	"github.com/jensneuse/graphql-go-tools/pkg/middleware"
	"github.com/jensneuse/graphql-go-tools/pkg/proxy"
)

func TestProxyHandler(t *testing.T) {
	es := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != assetOutput {
			t.Errorf("Expected %s, got %s", assetOutput, body)
		}
	}))
	defer es.Close()

	schemaProvider := proxy.NewStaticSchemaProvider([]byte(assetSchema))
	ip := sync.Pool{
		New: func() interface{} {
			return middleware.NewInvoker(&hackmiddleware.AssetUrlMiddleware{})
		},
	}
	ph := &Proxy{
		Host:           es.URL,
		SchemaProvider: schemaProvider,
		InvokerPool:    ip,
		Client:         *http.DefaultClient,
		HandleError: func(err error, w http.ResponseWriter) {
			t.Fatal(err)
		},
	}
	ts := httptest.NewServer(ph)
	defer ts.Close()

	t.Run("Test proxy handler", func(t *testing.T) {
		_, err := http.Post(ts.URL, "application/graphql", strings.NewReader(assetInput))
		if err != nil {
			t.Error(err)
		}
	})
}

func BenchmarkProxyHandler(b *testing.B) {
	es := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			b.Error(err)
		}
		if string(body) != assetOutput {
			b.Errorf("Expected %s, got %s", assetOutput, body)
		}
	}))
	defer es.Close()

	schemaProvider := proxy.NewStaticSchemaProvider([]byte(assetSchema))
	ip := sync.Pool{
		New: func() interface{} {
			return middleware.NewInvoker(&hackmiddleware.AssetUrlMiddleware{})
		},
	}
	ph := &Proxy{
		Host:           es.URL,
		SchemaProvider: schemaProvider,
		InvokerPool:    ip,
		Client:         *http.DefaultClient,
		HandleError: func(err error, w http.ResponseWriter) {
			b.Fatal(err)
		},
	}
	ts := httptest.NewServer(ph)
	defer ts.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := http.Post(ts.URL, "application/graphql", strings.NewReader(assetInput))
		if err != nil {
			b.Error(err)
		}
	}
}

const assetSchema = `
schema {
    query: Query
}

type Query {
    assets(first: Int): [Asset]
}

type Asset implements Node {
    status: Status!
    updatedAt: DateTime!
    createdAt: DateTime!
    id: ID!
    handle: String!
    fileName: String!
    height: Float
    width: Float
    size: Float
    mimeType: String
    url: String!
}`

const assetInput = `query testQueryWithoutHandle {
  								assets(first: 1) {
    							id
    							fileName
    							url(transformation: {image: {resize: {width: 100, height: 100}}})
  							}
						}`

const assetOutput = "query testQueryWithoutHandle {assets(first:1) {id fileName handle}}"
