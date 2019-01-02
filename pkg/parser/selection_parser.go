package parser

import (
	"github.com/jensneuse/graphql-go-tools/pkg/document"
	"github.com/jensneuse/graphql-go-tools/pkg/lexing/keyword"
)

func (p *Parser) parseSelection() (selection document.Selection, err error) {

	isFragmentSelection, err := p.peekExpect(keyword.SPREAD, true)
	if err != nil {
		return selection, err
	}

	if !isFragmentSelection {
		selection, err = p.parseField()
		return
	}

	isInlineFragment, err := p.peekExpect(keyword.ON, true)
	if err != nil {
		return selection, err
	}

	if isInlineFragment {
		selection, err = p.parseInlineFragment()
		return
	}

	selection, err = p.parseFragmentSpread()
	return
}
