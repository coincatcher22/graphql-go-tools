package resolve

import (
	"fmt"
	"sync"

	"github.com/buger/jsonparser"

	"github.com/jensneuse/graphql-go-tools/pkg/fastbuffer"
)

type DataLoader struct {
	fetches map[int]*FetchResult
	mu      sync.Mutex
}

func (d *DataLoader) LoadBatch(ctx *Context, batchFetch *BatchFetch) (response []byte, err error) {
	var fetchResult *FetchResult

	d.mu.Lock()
	fetchResult, ok := d.fetches[batchFetch.Fetch.BufferId]
	if ok {
		response, err = fetchResult.result(ctx)
	} else {
		fetchResult = &FetchResult{}
		d.fetches[batchFetch.Fetch.BufferId] = fetchResult
	}

	d.mu.Unlock()

	if err != nil {
		return nil, err
	}

	if response != nil {
		return response, nil
	}

	parentResponses, ok := d.fetches[ctx.lastFetchID]
	if !ok { // It must be impossible case
		return nil, fmt.Errorf("has not got fetch for %d", ctx.lastFetchID)
	}

	var inputs [][]byte
	buf := fastbuffer.New()

	for _, val := range d.selectedDataForFetch(parentResponses.results, ctx.responseElements...) {
		batchFetch.Fetch.InputTemplate.Render(ctx, val, buf)
		buf.Reset()
	}

	fetchResult.results, fetchResult.err = d.resolveBatchFetch(ctx, batchFetch, inputs...)

	return fetchResult.result(ctx)
}

func (d *DataLoader) resolveBatchFetch(ctx *Context, batchFetch *BatchFetch, inputs ...[]byte) (result [][]byte, err error) {
	batchInput, err := batchFetch.PrepareBatch(inputs...)
	if err != nil {
		return nil, err
	}

	fmt.Println("batch request", string(batchInput.Input))

	if ctx.beforeFetchHook != nil {
		ctx.beforeFetchHook.OnBeforeFetch(d.hookCtx(ctx), batchInput.Input)
	}

	batchBufferPair := &BufPair{
		Data:   fastbuffer.New(),
		Errors: fastbuffer.New(),
	}

	if err = batchFetch.Fetch.DataSource.Load(ctx.Context, batchInput.Input, batchBufferPair); err != nil {
		return nil, err
	}

	if ctx.afterFetchHook != nil {
		if batchBufferPair.HasData() {
			ctx.afterFetchHook.OnData(d.hookCtx(ctx), batchBufferPair.Data.Bytes(), false)
		}
		if batchBufferPair.HasErrors() {
			ctx.afterFetchHook.OnError(d.hookCtx(ctx), batchBufferPair.Errors.Bytes(), false)
		}
	}

	var outPosition int
	result = make([][]byte, 0, len(inputs))

	_, err = jsonparser.ArrayEach(batchBufferPair.Data.Bytes(), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		inputPositions := batchInput.OutToInPositions[outPosition]

		for _, pos := range inputPositions {
			result[pos] = value
		}

		outPosition++
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DataLoader) hookCtx(ctx *Context) HookContext {
	return HookContext{
		CurrentPath: ctx.path(),
	}
}

// @TODO fix performance
func (d *DataLoader) selectedDataForFetch(input [][]byte, path ...string) [][]byte {
	if len(path) == 0 {
		return input
	}

	current, rest := path[0], path[1:]

	if current == "@" {
		temp := make([][]byte, 0, len(input))

		for i := range input {
			var vals [][]byte
			jsonparser.ArrayEach(input[i], func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				vals = append(vals, value)
			})

			temp = append(temp, d.selectedDataForFetch(vals, rest...)...)
		}

		return temp
	}

	temp := make([][]byte, 0, len(input))

	for i := range input {
		el, _, _, err := jsonparser.Get(input[i], current)
		if err != nil {
			return nil
		}
		temp = append(temp, el)
	}

	return d.selectedDataForFetch(temp, rest...)
}

type FetchResult struct {
	nextIdx int
	err     error
	results [][]byte
}

func (f *FetchResult) result(ctx *Context) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}

	res := f.results[f.nextIdx]

	f.nextIdx++

	return res, nil
}
