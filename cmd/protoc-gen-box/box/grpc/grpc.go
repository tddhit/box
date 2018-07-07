package grpc

import (
	"fmt"
	"path"
	"strings"

	"github.com/tddhit/box/cmd/protoc-gen-box/generator"
	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

const generatedCodeVersion = 4

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	trPkgPath      = "github.com/tddhit/box/transport"
	trOptPkgPath   = "github.com/tddhit/box/transport/option"
	contextPkgPath = "golang.org/x/net/context"
	grpcPkgPath    = "google.golang.org/grpc"
)

func init() {
	generator.RegisterPlugin(new(grpc))
}

type grpc struct {
	gen *generator.Generator
}

func (g *grpc) Name() string {
	return "grpc"
}

var (
	trPkg      string
	trOptPkg   string
	contextPkg string
	grpcPkg    string
)

func (g *grpc) Init(gen *generator.Generator) {
	g.gen = gen
	trPkg = generator.RegisterUniquePackageName("tr", nil)
	trOptPkg = generator.RegisterUniquePackageName("tropt", nil)
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	grpcPkg = generator.RegisterUniquePackageName("grpc", nil)
}

func (g *grpc) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

func (g *grpc) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

func (g *grpc) P(args ...interface{}) { g.gen.P(args...) }

func (g *grpc) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ ", trPkg, ".Server")
	g.P("var _ ", trPkg, ".ClientConn")
	g.P("var _ ", trOptPkg, ".CallOption")
	g.P()

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *grpc) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("import (")
	g.P(trPkg, " ", generator.GoImportPath(path.Join(string(g.gen.ImportPrefix), trPkgPath)))
	g.P(trOptPkg, " ", generator.GoImportPath(path.Join(string(g.gen.ImportPrefix), trOptPkgPath)))
	g.P(contextPkg, " ", generator.GoImportPath(path.Join(string(g.gen.ImportPrefix), contextPkgPath)))
	g.P(grpcPkg, " ", generator.GoImportPath(path.Join(string(g.gen.ImportPrefix), grpcPkgPath)))
	g.P(")")
	g.P()
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
func (g *grpc) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)

	// Box Client interface.
	g.P("type ", servName, "GrpcClient interface {")
	for _, method := range service.Method {
		g.P(g.generateClientSignature(servName, method))
	}
	g.P("}")
	g.P()

	// Box Client structure.
	g.P("type ", unexport(servName), "GrpcClient struct {")
	g.P("cc ", trPkg, ".ClientConn")
	g.P("}")
	g.P()

	// Box NewClient factory.
	g.P("func New", servName, "GrpcClient (cc ", trPkg,
		".ClientConn) ", servName, "GrpcClient {")
	g.P("return &", unexport(servName), "GrpcClient{cc}")
	g.P("}")
	g.P()

	serviceDescVar := "_" + servName + "_serviceDesc"
	for _, method := range service.Method {
		g.generateClientMethod(servName, fullServName,
			serviceDescVar, method)
	}

	g.P("type ", unexport(servName), "GrpcServiceDesc struct {")
	g.P("desc *", grpcPkg, ".ServiceDesc")
	g.P("}")
	g.P()

	g.P("func (d *", unexport(servName),
		"GrpcServiceDesc) Desc() interface{} {")
	g.P("return d.desc")
	g.P("}")
	g.P()

	g.P("var ", servName, "GrpcServiceDesc = ", "&",
		unexport(servName), "GrpcServiceDesc{&", serviceDescVar, "}")
}

// generateClientSignature returns the client-side signature for a method.
func (g *grpc) generateClientSignature(servName string,
	method *pb.MethodDescriptorProto) string {

	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + g.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "*" + g.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx %s.Context%s, opts ...%s.CallOption) (%s, error)", methName, contextPkg, reqArg, trOptPkg, respName)
}

func (g *grpc) generateClientMethod(servName, fullServName,
	serviceDescVar string, method *pb.MethodDescriptorProto) {

	sname := fmt.Sprintf("/%s/%s", fullServName, method.GetName())
	outType := g.typeName(method.GetOutputType())

	if method.GetOptions().GetDeprecated() {
		g.P(deprecationComment)
	}
	g.P("func (c *", unexport(servName), "GrpcClient) ",
		g.generateClientSignature(servName, method), "{")

	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		g.P("out := new(", outType, ")")
		// TODO: Pass descExpr to Invoke.
		g.P(`err := c.cc.Invoke(ctx, "`, sname, `", in, out, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	}
}
