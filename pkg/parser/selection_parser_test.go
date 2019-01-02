package parser

import (
	. "github.com/franela/goblin"
	"github.com/jensneuse/graphql-go-tools/pkg/document"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"testing"
)

func TestSelectionParser(t *testing.T) {

	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("parser.parseSelection", func() {

		tests := []struct {
			it           string
			input        string
			expectErr    types.GomegaMatcher
			expectValues types.GomegaMatcher
		}{
			{
				it:        "should parse a InlineFragment",
				input:     "...on Land",
				expectErr: BeNil(),
				expectValues: Equal(document.InlineFragment{
					TypeCondition: document.NamedType{
						Name: "Land",
					},
				}),
			},
			{
				it:        "should parse a simple Field",
				input:     "originalName",
				expectErr: BeNil(),
				expectValues: Equal(document.Field{
					Name: "originalName",
				}),
			},
			{
				it:        "should parse a nested selection",
				input:     `t { kind name ofType { kind name ofType { kind name } } }`,
				expectErr: BeNil(),
				expectValues: Equal(document.Field{
					Name: "t",
					SelectionSet: []document.Selection{
						document.Field{
							Name: "kind",
						},
						document.Field{
							Name: "name",
						},
						document.Field{
							Name: "ofType",
							SelectionSet: []document.Selection{
								document.Field{
									Name: "kind",
								},
								document.Field{
									Name: "name",
								},
								document.Field{
									Name: "ofType",
									SelectionSet: []document.Selection{
										document.Field{
											Name: "kind",
										},
										document.Field{
											Name: "name",
										},
									},
								},
							},
						},
					},
				}),
			},
			{
				it:        "should parse a simple Field with an argument",
				input:     "originalName(isSet: true)",
				expectErr: BeNil(),
				expectValues: Equal(document.Field{
					Name: "originalName",
					Arguments: document.Arguments{
						document.Argument{
							Name: "isSet",
							Value: document.BooleanValue{
								Val: true,
							},
						},
					},
				}),
			},
			{
				it:        "should parse a FragmentSpread",
				input:     "...Land",
				expectErr: BeNil(),
				expectValues: Equal(document.FragmentSpread{
					FragmentName: "Land",
				}),
			},
		}

		for _, test := range tests {
			test := test

			g.It(test.it, func() {

				parser := NewParser()
				parser.l.SetInput(test.input)

				val, err := parser.parseSelection()
				Expect(err).To(test.expectErr)
				Expect(val).To(test.expectValues)
			})
		}
	})
}

var parseSelectionBenchmarkInput = `t { kind name ofType { kind name ofType { kind name } } }`

func BenchmarkParseSelection(b *testing.B) {

	parser := NewParser()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {

		parser.l.SetInput(parseSelectionBenchmarkInput)
		_, err := parser.parseSelection()
		if err != nil {
			b.Fatal(err)
		}
	}
}
