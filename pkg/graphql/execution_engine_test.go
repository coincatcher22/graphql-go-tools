package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jensneuse/abstractlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jensneuse/graphql-go-tools/pkg/execution/datasource"
	"github.com/jensneuse/graphql-go-tools/pkg/starwars"
)

type testRoundTripper func(req *http.Request) *http.Response

func (t testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t(req), nil
}

func TestExecutionEngine_ExecuteWithOptions(t *testing.T) {
	type testCase struct {
		schema            *Schema
		request           func(t *testing.T) Request
		plannerConfig     datasource.PlannerConfiguration
		roundTripper      testRoundTripper
		preExecutionTasks func(t *testing.T, request Request, schema *Schema, engine *ExecutionEngine) // optional
		expectedResponse  string
	}

	loadStarWarsQuery := func(t *testing.T, starwarsFile string, variables starwars.QueryVariables) func(t *testing.T) Request {
		return func(t *testing.T) Request {
			query := starwars.LoadQuery(t, starwarsFile, variables)
			request := Request{}
			err := UnmarshalRequest(bytes.NewBuffer(query), &request)
			require.NoError(t, err)

			return request
		}
	}

	createTestRoundTripper := func(host string, url string, response string, statusCode int) testRoundTripper {
		return testRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, host, req.URL.Host)
			assert.Equal(t, url, req.URL.Path)

			body := bytes.NewBuffer([]byte(response))
			return &http.Response{StatusCode: statusCode, Body: ioutil.NopCloser(body)}
		})
	}

	run := func(tc testCase, hasExecutionError bool) func(t *testing.T) {
		return func(t *testing.T) {
			request := tc.request(t)

			extraVariables := map[string]string{
				"request": `{"Authorization": "Bearer ey123"}`,
			}
			extraVariablesBytes, err := json.Marshal(extraVariables)
			require.NoError(t, err)

			engine, err := NewExecutionEngine(abstractlogger.NoopLogger, tc.schema, tc.plannerConfig)
			assert.NoError(t, err)

			switch tc.plannerConfig.TypeFieldConfigurations[0].DataSource.Name {
			case "HttpJsonDataSource":
				httpJsonOptions := DataSourceHttpJsonOptions{
					HttpClient: &http.Client{
						Transport: tc.roundTripper,
					},
				}

				err = engine.AddHttpJsonDataSourceWithOptions("HttpJsonDataSource", httpJsonOptions)
				assert.NoError(t, err)
			case "GraphqlDataSource":
				graphqlOptions := DataSourceGraphqlOptions{
					HttpClient: &http.Client{
						Transport: tc.roundTripper,
					},
				}

				err = engine.AddGraphqlDataSourceWithOptions("GraphqlDataSource", graphqlOptions)
				assert.NoError(t, err)
			}

			if tc.preExecutionTasks != nil {
				tc.preExecutionTasks(t, request, tc.schema, engine)
			}

			executionRes, err := engine.Execute(context.Background(), &request, ExecutionOptions{ExtraArguments: extraVariablesBytes})

			if hasExecutionError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedResponse, executionRes.Buffer().String())
		}
	}

	runWithoutError := func(tc testCase) func(t *testing.T) {
		return run(tc, false)
	}

	runWithError := func(tc testCase) func(t *testing.T) {
		return run(tc, true)
	}

	t.Run("execute with custom roundtripper for simple hero query on HttpJsonDatasource", runWithoutError(testCase{
		schema:           starwarsSchema(t),
		request:          loadStarWarsQuery(t, starwars.FileSimpleHeroQuery, nil),
		plannerConfig:    heroHttpJsonPlannerConfig,
		roundTripper:     createTestRoundTripper("example.com", "/", `{"hero": {"name": "Luke Skywalker"}}`, 200),
		expectedResponse: `{"data":{"hero":{"name":"Luke Skywalker"}}}`,
	}))

	t.Run("execute with empty request object should not panic", runWithError(testCase{
		schema: starwarsSchema(t),
		request: func(t *testing.T) Request {
			return Request{}
		},
		plannerConfig:    heroHttpJsonPlannerConfig,
		roundTripper:     createTestRoundTripper("example.com", "/", `{"hero": {"name": "Luke Skywalker"}}`, 200),
		expectedResponse: "",
	}))

	t.Run("execute with empty request object should not panic", runWithoutError(testCase{
		schema:           starwarsSchema(t),
		request:          loadStarWarsQuery(t, starwars.FileSimpleHeroQuery, nil),
		plannerConfig:    heroGraphqlDataSource,
		roundTripper:     createTestRoundTripper("example.com", "/", `{"data":{"hero":{"name":"Luke Skywalker"}}}`, 200),
		expectedResponse: `{"data":{"hero":{"name":"Luke Skywalker"}}}`,
	}))

	t.Run("execute with empty request object should not panic", runWithoutError(testCase{
		schema: starwarsSchema(t),
		request: func(t *testing.T) Request {
			request := loadStarWarsQuery(t, starwars.FileMultiQueries, nil)(t)
			request.OperationName = "SingleHero"
			return request
		},
		plannerConfig:    heroGraphqlDataSource,
		roundTripper:     createTestRoundTripper("example.com", "/", `{"data":{"hero":{"name":"Luke Skywalker"}}}`, 200),
		expectedResponse: `{"data":{"hero":{"name":"Luke Skywalker"}}}`,
	}))

	t.Run("execute query with variables for arguments", runWithoutError(testCase{
		schema:            starwarsSchema(t),
		request:           loadStarWarsQuery(t, starwars.FileDroidWithArgAndVarQuery, map[string]interface{}{"droidID": "R2D2"}),
		plannerConfig:     droidGraphqlDataSource,
		roundTripper:      createTestRoundTripper("example.com", "/", `{"data":{"droid":{"name":"R2D2"}}}`, 200),
		preExecutionTasks: normalizeAndValidatePreExecutionTasks,
		expectedResponse:  `{"data":{"droid":{"name":"R2D2"}}}`,
	}))

	t.Run("execute query with arguments", runWithoutError(testCase{
		schema:            starwarsSchema(t),
		request:           loadStarWarsQuery(t, starwars.FileDroidWithArgQuery, nil),
		plannerConfig:     droidGraphqlDataSource,
		roundTripper:      createTestRoundTripper("example.com", "/", `{"data":{"droid":{"name":"R2D2"}}}`, 200),
		preExecutionTasks: normalizeAndValidatePreExecutionTasks,
		expectedResponse:  `{"data":{"droid":{"name":"R2D2"}}}`,
	}))

	t.Run("execute single mutation with arguments on document with multiple operations", runWithoutError(testCase{
		schema: moviesSchema(t),
		request: func(t *testing.T) Request {
			return Request{
				OperationName: "AddWithInput",
				Variables:     nil,
				Query: `mutation AddToWatchlist {
						  addToWatchlist(movieID:3) {
							id
							name
							year
						  }
						}
						
						
						mutation AddWithInput {
						  addToWatchlistWithInput(input: {id: 2}) {
							id
							name
							year
						  }
						}`,
			}
		},
		plannerConfig:     movieHttpJsonDataSource,
		roundTripper:      createTestRoundTripper("example.com", "/", `{"added_movie":{"id":2, "name": "Episode V – The Empire Strikes Back", "year": 1980}}`, 200),
		preExecutionTasks: normalizeAndValidatePreExecutionTasks,
		expectedResponse:  `{"data":{"addToWatchlistWithInput":{"id":2,"name":"Episode V – The Empire Strikes Back","year":1980}}}`,
	}))
}

func normalizeAndValidatePreExecutionTasks(t *testing.T, request Request, schema *Schema, engine *ExecutionEngine) {
	normalizationResult, err := request.Normalize(schema)
	require.NoError(t, err)
	require.True(t, normalizationResult.Successful)

	validationResult, err := request.ValidateForSchema(schema)
	require.NoError(t, err)
	require.True(t, validationResult.Valid)
}

func stringify(any interface{}) []byte {
	out, _ := json.Marshal(any)
	return out
}

func stringPtr(str string) *string {
	return &str
}

func moviesSchema(t *testing.T) *Schema {
	schemaString := `
type Movie {
  id: Int!
  name: String!
  year: Int!
}

type Mutation {
  addToWatchlist(movieID: Int!): Movie
  addToWatchlistWithInput(input: WatchlistInput!): Movie
}

type Query {
  default: String
}

input WatchlistInput {
  id: Int!
}`
	schema, err := NewSchemaFromString(schemaString)
	require.NoError(t, err)
	return schema
}

func TestExampleExecutionEngine_Concatenation(t *testing.T) {

	schema, err := NewSchemaFromString(`
		schema { query: Query }
		type Query { friend: Friend }
		type Friend { firstName: String lastName: String fullName: String }
	`)

	assert.NoError(t, err)

	friendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"firstName":"Jens","lastName":"Neuse"}`))
	}))

	defer friendServer.Close()

	pipelineConcat := `
	{
		"steps": [
			{
				"kind": "JSON",
				"config": {
					"template": "{\"fullName\":\"{{ .firstName }} {{ .lastName }}\"}"
				}
			}
  		]
	}`

	plannerConfig := datasource.PlannerConfiguration{
		TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
			{
				TypeName:  "query",
				FieldName: "friend",
				Mapping: &datasource.MappingConfiguration{
					Disabled: true,
				},
				DataSource: datasource.SourceConfig{
					Name: "HttpJsonDataSource",
					Config: stringify(datasource.HttpJsonDataSourceConfig{
						Host:   friendServer.URL,
						Method: stringPtr("GET"),
					}),
				},
			},
			{
				TypeName:  "Friend",
				FieldName: "fullName",
				DataSource: datasource.SourceConfig{
					Name: "FriendFullName",
					Config: stringify(datasource.PipelineDataSourceConfig{
						ConfigString: stringPtr(pipelineConcat),
						InputJSON:    `{"firstName":"{{ .object.firstName }}","lastName":"{{ .object.lastName }}"}`,
					}),
				},
			},
		},
	}

	engine, err := NewExecutionEngine(abstractlogger.NoopLogger, schema, plannerConfig)
	assert.NoError(t, err)
	err = engine.AddHttpJsonDataSource("HttpJsonDataSource")
	assert.NoError(t, err)
	err = engine.AddDataSource("FriendFullName", datasource.PipelineDataSourcePlannerFactoryFactory{})
	assert.NoError(t, err)

	request := &Request{
		Query: `query { friend { firstName lastName fullName }}`,
	}

	executionRes, err := engine.Execute(context.Background(), request, ExecutionOptions{})
	assert.NoError(t, err)

	expected := `{"data":{"friend":{"firstName":"Jens","lastName":"Neuse","fullName":"Jens Neuse"}}}`
	actual := executionRes.Buffer().String()
	assert.Equal(t, expected, actual)
}

func BenchmarkExecutionEngine(b *testing.B) {

	newEngine := func() *ExecutionEngine {
		schema, err := NewSchemaFromString(`type Query { hello: String}`)
		assert.NoError(b, err)
		plannerConfig := datasource.PlannerConfiguration{
			TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
				{
					TypeName:  "query",
					FieldName: "hello",
					Mapping: &datasource.MappingConfiguration{
						Disabled: true,
					},
					DataSource: datasource.SourceConfig{
						Name: "HelloDataSource",
						Config: stringify(datasource.StaticDataSourceConfig{
							Data: "world",
						}),
					},
				},
			},
		}
		engine, err := NewExecutionEngine(abstractlogger.NoopLogger, schema, plannerConfig)
		assert.NoError(b, err)
		assert.NoError(b, engine.AddDataSource("HelloDataSource", datasource.StaticDataSourcePlannerFactoryFactory{}))
		return engine
	}

	ctx := context.Background()
	req := &Request{
		Query: "{hello}",
	}
	out := bytes.Buffer{}
	err := newEngine().ExecuteWithWriter(ctx, req, &out, ExecutionOptions{})
	assert.NoError(b, err)
	assert.Equal(b, "{\"data\":{\"hello\":\"world\"}}", out.String())

	pool := sync.Pool{
		New: func() interface{} {
			return newEngine()
		},
	}

	b.SetBytes(int64(out.Len()))
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine := pool.Get().(*ExecutionEngine)
			_ = engine.ExecuteWithWriter(ctx, req, ioutil.Discard, ExecutionOptions{})
			pool.Put(engine)
		}
	})
}

var heroHttpJsonPlannerConfig = datasource.PlannerConfiguration{
	TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
		{
			TypeName:  "Query",
			FieldName: "hero",
			Mapping: &datasource.MappingConfiguration{
				Disabled: false,
				Path:     "hero",
			},
			DataSource: datasource.SourceConfig{
				Name: "HttpJsonDataSource",
				Config: func() []byte {
					data, _ := json.Marshal(datasource.HttpJsonDataSourceConfig{
						Host: "example.com",
						URL:  "/",
						Method: func() *string {
							method := "GET"
							return &method
						}(),
						DefaultTypeName: func() *string {
							typeName := "Hero"
							return &typeName
						}(),
					})
					return data
				}(),
			},
		},
	},
}

var movieHttpJsonDataSource = datasource.PlannerConfiguration{
	TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
		{
			TypeName:  "Mutation",
			FieldName: "addToWatchlistWithInput",
			Mapping: &datasource.MappingConfiguration{
				Disabled: false,
				Path:     "added_movie",
			},
			DataSource: datasource.SourceConfig{
				Name: "HttpJsonDataSource",
				Config: func() []byte {
					data, _ := json.Marshal(datasource.HttpJsonDataSourceConfig{
						Host: "example.com",
						URL:  "/",
						Method: func() *string {
							method := "GET"
							return &method
						}(),
						DefaultTypeName: func() *string {
							typeName := "Movie"
							return &typeName
						}(),
					})
					return data
				}(),
			},
		},
	},
}

var heroGraphqlDataSource = datasource.PlannerConfiguration{
	TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
		{
			TypeName:  "query",
			FieldName: "hero",
			Mapping: &datasource.MappingConfiguration{
				Disabled: false,
				Path:     "hero",
			},
			DataSource: datasource.SourceConfig{
				Name: "GraphqlDataSource",
				Config: func() []byte {
					data, _ := json.Marshal(datasource.GraphQLDataSourceConfig{
						Host: "example.com",
						URL:  "/",
						Method: func() *string {
							method := "GET"
							return &method
						}(),
					})
					return data
				}(),
			},
		},
	},
}

var droidGraphqlDataSource = datasource.PlannerConfiguration{
	TypeFieldConfigurations: []datasource.TypeFieldConfiguration{
		{
			TypeName:  "query",
			FieldName: "droid",
			Mapping: &datasource.MappingConfiguration{
				Disabled: false,
				Path:     "droid",
			},
			DataSource: datasource.SourceConfig{
				Name: "GraphqlDataSource",
				Config: func() []byte {
					data, _ := json.Marshal(datasource.GraphQLDataSourceConfig{
						Host: "example.com",
						URL:  "/",
						Method: func() *string {
							method := "GET"
							return &method
						}(),
					})
					return data
				}(),
			},
		},
	},
}
