package parser

import (
	"bytes"
	. "github.com/franela/goblin"
	"github.com/jensneuse/graphql-go-tools/pkg/document"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"testing"
)

func TestDefaultValueParser(t *testing.T) {

	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("parser.parseDefaultValue", func() {

		tests := []struct {
			it           string
			input        string
			expectErr    types.GomegaMatcher
			expectValues types.GomegaMatcher
		}{
			{
				it:        "should parse a simple DefaultValue",
				input:     "= 2",
				expectErr: BeNil(),
				expectValues: Equal(document.IntValue{
					Val: 2,
				}),
			},
			{
				it:           "should ignore a non existing DefaultValue",
				input:        " ",
				expectErr:    BeNil(),
				expectValues: BeNil(),
			},
			{
				it:           "should not parse when no EQUALS is set",
				input:        "2",
				expectErr:    BeNil(),
				expectValues: BeNil(),
			},
		}

		for _, test := range tests {
			test := test

			g.It(test.it, func() {

				reader := bytes.NewReader([]byte(test.input))
				parser := NewParser()
				parser.l.SetInput(reader)

				val, err := parser.parseDefaultValue()
				Expect(err).To(test.expectErr)
				Expect(val).To(test.expectValues)
			})
		}
	})
}
