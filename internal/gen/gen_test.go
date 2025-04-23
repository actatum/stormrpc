package gen

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateFiles(t *testing.T) {
	t.Run("single_file", func(t *testing.T) {
		plugin := parseFilesIntoRequest(t, []string{"single_file.proto"})

		GenerateFiles(plugin)

		resp := plugin.Response()

		if len(resp.File) != 1 {
			t.Fatal("expected only one file to be generated")
		}

		goldenFile, err := os.ReadFile("./testdata/single_file.golden")
		if err != nil {
			t.Fatal("reading golden file: %w", err)
		}

		diff := cmp.Diff(
			resp.File[0].GetContent(),
			string(goldenFile),
		)
		if diff != "" {
			t.Fatalf("diff: %s", diff)
		}
	})

	t.Run("multiple_file", func(t *testing.T) {
		plugin := parseFilesIntoRequest(t, []string{"multiple_file_1.proto", "multiple_file_2.proto"})

		GenerateFiles(plugin)

		resp := plugin.Response()

		if len(resp.File) != 2 {
			t.Fatal("expected two files to be generated")
		}

		goldenFile1, err := os.ReadFile("./testdata/multiple_file_1.golden")
		if err != nil {
			t.Fatal("reading golden file 1: %w", err)
		}

		diff := cmp.Diff(
			resp.File[0].GetContent(),
			string(goldenFile1),
		)
		if diff != "" {
			t.Fatalf("diff_1: %s", diff)
		}

		goldenFile2, err := os.ReadFile("./testdata/multiple_file_2.golden")
		if err != nil {
			t.Fatal("reading golden file 2: %w", err)
		}

		diff = cmp.Diff(
			resp.File[1].GetContent(),
			string(goldenFile2),
		)
		if diff != "" {
			t.Fatalf("diff_2: %s", diff)
		}
	})

	t.Run("multiple_package", func(t *testing.T) {
		plugin := parseFilesIntoRequest(t, []string{"multiple_package_1.proto", "multiple_package_2.proto"})

		GenerateFiles(plugin)

		resp := plugin.Response()

		if len(resp.File) != 2 {
			t.Fatal("expected two files to be generated")
		}

		goldenFile1, err := os.ReadFile("./testdata/multiple_package_1.golden")
		if err != nil {
			t.Fatal("reading golden file 1: %w", err)
		}

		diff := cmp.Diff(
			resp.File[0].GetContent(),
			string(goldenFile1),
		)
		if diff != "" {
			t.Fatalf("diff_1: %s", diff)
		}

		goldenFile2, err := os.ReadFile("./testdata/multiple_package_2.golden")
		if err != nil {
			t.Fatal("reading golden file 2: %w", err)
		}

		diff = cmp.Diff(
			resp.File[1].GetContent(),
			string(goldenFile2),
		)
		if diff != "" {
			t.Fatalf("diff_2: %s", diff)
		}
	})
}

func parseFilesIntoRequest(t *testing.T, fileNames []string) *protogen.Plugin {
	t.Helper()

	parser := protoparse.Parser{
		ImportPaths: []string{"./testdata"},
	}

	descs, err := parser.ParseFiles(fileNames...)
	if err != nil {
		t.Fatal("parsing proto files: %w", err)
	}

	filesToGenerate := make([]string, 0)

	for _, desc := range descs {
		fdProto := desc.AsFileDescriptorProto()
		filesToGenerate = append(filesToGenerate, fdProto.GetName())
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: filesToGenerate,
		ProtoFile:      []*descriptorpb.FileDescriptorProto{},
	}

	for _, desc := range descs {
		req.ProtoFile = append(req.ProtoFile, desc.AsFileDescriptorProto())
	}

	opts := protogen.Options{}
	genPlugin, err := opts.New(req)
	if err != nil {
		t.Fatal("creating plugin: %w", err)
	}

	return genPlugin
}
