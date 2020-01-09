package execution

import (
	"bytes"
	"github.com/jensneuse/graphql-go-tools/pkg/lexer/literal"
	"github.com/jensneuse/pipeline/pkg/pipeline"
	"go.uber.org/zap"
	"os"
	"testing"
)

func TestPipelineDataSource_Resolve(t *testing.T) {

	configFile,err := os.Open("./testdata/simple_pipeline.json")
	if err != nil {
		t.Fatal(err)
	}

	defer configFile.Close()

	var pipe pipeline.Pipeline
	err = pipe.FromConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}

	source := PipelineDataSource{
		log:zap.NewNop(),
		pipe: pipe,
	}

	args := []ResolvedArgument{
		{
			Key:   literal.INPUT_JSON,
			Value: []byte(`{"foo":"bar"}`),
		},
	}

	var out bytes.Buffer
	source.Resolve(Context{},args,&out)

	got := out.String()
	want := `{"foo":"bar"}`

	if want != got {
		t.Fatalf("want: %s\ngot: %s\n",want,got)
	}
}
