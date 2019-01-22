package parser

import (
	"github.com/jensneuse/graphql-go-tools/pkg/document"
	"github.com/jensneuse/graphql-go-tools/pkg/lexing/keyword"
)

func (p *Parser) parseSelectionSet(set *document.SelectionSet) (err error) {

	if open := p.peekExpect(keyword.CURLYBRACKETOPEN, false); !open {
		return
	}

	start := p.l.Read()
	set.Position.MergeStartIntoStart(start.TextPosition)

	for {

		next := p.l.Peek(true)

		if next == keyword.CURLYBRACKETCLOSE {
			end := p.l.Read()
			set.Position.MergeEndIntoEnd(end.TextPosition)
			return nil
		}

		isFragmentSelection := p.peekExpect(keyword.SPREAD, false)
		if !isFragmentSelection {
			err := p.parseField(&set.Fields)
			if err != nil {
				return err
			}
		} else {

			start := p.l.Read()

			isInlineFragment := p.peekExpect(keyword.ON, true)
			if isInlineFragment {

				err := p.parseInlineFragment(start.TextPosition, &set.InlineFragments)
				if err != nil {
					return err
				}

			} else {

				err := p.parseFragmentSpread(start.TextPosition, &set.FragmentSpreads)
				if err != nil {
					return err
				}
			}
		}
	}
}
