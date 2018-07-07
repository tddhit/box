package http

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/httprule"
	options "google.golang.org/genproto/googleapis/api/annotations"

	"github.com/tddhit/box/cmd/protoc-gen-box/generator"
	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

const generatedCodeVersion = 4

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "golang.org/x/net/context"
	runtimePkgPath = "github.com/grpc-ecosystem/grpc-gateway/runtime"
	trPkgPath      = "github.com/tddhit/box/transport"
	trhttpPkgPath  = "github.com/tddhit/box/transport/http"
	troptPkgPath   = "github.com/tddhit/box/transport/option"
)

func init() {
	generator.RegisterPlugin(new(http))
}

type http struct {
	gen *generator.Generator
}

func (h *http) Name() string {
	return "http"
}

var (
	contextPkg string
	runtimePkg string
	trPkg      string
	trhttpPkg  string
	troptPkg   string
)

func (h *http) Init(gen *generator.Generator) {
	h.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	runtimePkg = generator.RegisterUniquePackageName("runtime", nil)
	trPkg = generator.RegisterUniquePackageName("tr", nil)
	trhttpPkg = generator.RegisterUniquePackageName("trhttp", nil)
	troptPkg = generator.RegisterUniquePackageName("tropt", nil)
}

func (h *http) objectNamed(name string) generator.Object {
	h.gen.RecordTypeUse(name)
	return h.gen.ObjectNamed(name)
}

func (h *http) typeName(str string) string {
	return h.gen.TypeName(h.objectNamed(str))
}

func (h *http) P(args ...interface{}) { h.gen.P(args...) }

func (h *http) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	for i, service := range file.FileDescriptorProto.Service {
		h.generateService(file, service, i)
	}
}

func (h *http) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	h.P("import (")
	h.P(contextPkg, " ", generator.GoImportPath(path.Join(string(h.gen.ImportPrefix), contextPkgPath)))
	h.P(runtimePkg, " ", generator.GoImportPath(path.Join(string(h.gen.ImportPrefix), runtimePkgPath)))
	h.P(trPkg, " ", generator.GoImportPath(path.Join(string(h.gen.ImportPrefix), trPkgPath)))
	h.P(trhttpPkg, " ", generator.GoImportPath(path.Join(string(h.gen.ImportPrefix), trhttpPkgPath)))
	h.P(troptPkg, " ", generator.GoImportPath(path.Join(string(h.gen.ImportPrefix), troptPkgPath)))
	h.P(")")
	h.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
	// TODO: do we need any in gRPC?
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// deprecationComment is the standard comment added to deprecated
// messages, fields, enums, and enum values.
var deprecationComment = "// Deprecated: Do not use."

// generateService generates all the code for the named service.
func (h *http) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)

	// Box Client interface.
	h.P("type ", servName, "HttpClient interface {")
	for _, method := range service.Method {
		h.P(h.generateClientSignature(servName, method))
	}
	h.P("}")
	h.P()

	// Box Client structure.
	h.P("type ", unexport(servName), "HttpClient struct {")
	h.P("cc ", trPkg, ".ClientConn")
	h.P("}")
	h.P()

	// Box NewClient factory.
	h.P("func New", servName, "HttpClient (cc ", trPkg,
		".ClientConn) ", servName, "HttpClient {")
	h.P("return &", unexport(servName), "HttpClient{cc}")
	h.P("}")
	h.P()

	serviceDescVar := "_" + servName + "_Http_serviceDesc"
	for _, method := range service.Method {
		h.generateClientMethod(servName,
			fullServName, serviceDescVar, method)
	}

	h.P("var ", serviceDescVar, " = ", trhttpPkg, ".ServiceDesc {")
	h.P("ServiceDesc: ", "&_", servName, "_serviceDesc,")
	h.P("Pattern: map[string]", runtimePkg, ".Pattern{")
	for _, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		pattern := fmt.Sprintf("pattern_%s_%s",
			servName, method.GetName())
		h.P(strconv.Quote(method.GetName()), ": ", pattern, ",")
	}
	h.P("},")
	h.P("}")
	h.P()

	h.P("type ", unexport(servName), "HttpServiceDesc struct {")
	h.P("desc *", trhttpPkg, ".ServiceDesc")
	h.P("}")
	h.P()

	h.P("func (d *", unexport(servName),
		"HttpServiceDesc) Desc() interface{} {")
	h.P("return d.desc")
	h.P("}")
	h.P()

	h.P("var ", servName, "HttpServiceDesc = ", "&",
		unexport(servName), "HttpServiceDesc{&", serviceDescVar, "}")
	h.P()

	h.P("var (")
	for _, method := range service.Method {
		ext, _ := proto.GetExtension(method.Options, options.E_Http)
		opts, _ := ext.(*options.HttpRule)
		pathTemplate := opts.GetPost()
		parsed, _ := httprule.Parse(pathTemplate)
		tmpl := parsed.Compile()
		pattern := fmt.Sprintf("pattern_%s_%s",
			servName, method.GetName())
		h.P(pattern, " = ", runtimePkg, ".MustPattern(",
			runtimePkg, ".NewPattern(", tmpl.Version, ", ",
			fmt.Sprintf("%#v", tmpl.OpCodes), ", ",
			fmt.Sprintf("%#v", tmpl.Pool), ", ",
			`""))`,
		)
	}
	h.P(")")
	h.P()
}

// generateClientSignature returns the client-side signature for a method.
func (h *http) generateClientSignature(servName string,
	method *pb.MethodDescriptorProto) string {

	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + h.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "*" + h.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx %s.Context%s, opts ...%s.CallOption) (%s, error)", methName, contextPkg, reqArg, troptPkg, respName)
}

func (h *http) generateClientMethod(servName string, fullServName string,
	serviceDescVar string, method *pb.MethodDescriptorProto) {

	outType := h.typeName(method.GetOutputType())
	if method.GetOptions().GetDeprecated() {
		h.P(deprecationComment)
	}
	h.P("func (c *", unexport(servName), "HttpClient) ",
		h.generateClientSignature(servName, method), "{")
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		h.P("out := new(", outType, ")")
		h.P("pattern := ", serviceDescVar, ".Pattern[",
			strconv.Quote(method.GetName()), "]")
		h.P("err := c.cc.Invoke(ctx, pattern.String(), in, out, opts...)")
		h.P("if err != nil { return nil, err }")
		h.P("return out, nil")
		h.P("}")
		h.P()
		return
	}
}
