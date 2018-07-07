package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/urfave/cli"

	"github.com/tddhit/tools/log"
)

type Generator struct {
	Project     string
	CCProject   string
	PkgPath     string
	GoPath      string
	UseCC       bool
	UseGRPC     bool
	UseHTTP     bool
	UseGW       bool
	UseRegistry bool

	mainTmpl    *template.Template
	pbTmpl      *template.Template
	confTmpl    *template.Template
	serviceTmpl *template.Template
	clientTmpl  *template.Template
	gwTmpl      *template.Template
}

func NewGenerator() *Generator {
	g := &Generator{}
	g.mainTmpl = template.Must(
		template.New("main").Funcs(
			template.FuncMap{
				"RunOptionList": g.RunOptionList,
			},
		).Parse(mainCode),
	)
	g.pbTmpl = template.Must(
		template.New("pb").Funcs(
			template.FuncMap{},
		).Parse(pbCode),
	)
	g.confTmpl = template.Must(
		template.New("conf").Funcs(
			template.FuncMap{},
		).Parse(confCode),
	)
	g.serviceTmpl = template.Must(
		template.New("service").Funcs(
			template.FuncMap{},
		).Parse(serviceCode),
	)
	g.clientTmpl = template.Must(
		template.New("client").Funcs(
			template.FuncMap{},
		).Parse(clientCode),
	)
	return g
}

func (g *Generator) protoc(out string) {
	includePath1 := fmt.Sprintf("%s/src/%s/pb", g.GoPath, g.PkgPath)
	includePath2 := fmt.Sprintf(
		"%s/src/github.com/tddhit/box/third_party/googleapis",
		g.GoPath)

	cmd := exec.Command("protoc", "-I", includePath1,
		"-I", includePath2, out, g.Project+".proto")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		log.Error(buf.String())
		log.Panic(err)
	}
}

func (g *Generator) execProtoc() {
	plugin := "plugins="
	if g.UseGRPC {
		plugin += "grpc+"
	}
	if g.UseHTTP {
		plugin += "http+"
	}
	plugin = strings.TrimSuffix(plugin, "+")
	out := fmt.Sprintf("--box_out=%s:%s/src/%s/pb",
		plugin, g.GoPath, g.PkgPath)
	g.protoc(out)

	if g.UseGW {
		out := fmt.Sprintf("--grpc-gateway_out=%s/src/%s/pb",
			g.GoPath, g.PkgPath)
		g.protoc(out)
	}
}

func (g *Generator) GenPB() {
	if !g.UseGRPC {
		return
	}
	os.MkdirAll(g.Project+"/pb", 0755)
	filename := fmt.Sprintf("%s/src/%s/pb/%s.proto",
		g.GoPath, g.PkgPath, g.Project)
	g.generate(g.pbTmpl, filename)
	g.execProtoc()
}

func (g *Generator) GenConf() {
	os.MkdirAll(g.Project+"/conf", 0755)
	filename := fmt.Sprintf("%s/conf/conf.go", g.Project)
	g.generate(g.confTmpl, filename)

	filename = fmt.Sprintf("%s/conf/%s.yml", g.Project, g.Project)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	f.WriteString(`logpath: ""`)
	f.WriteString("\nloglevel: 1 # Debug(1)、Info(2) ...")
	f.Sync()
	f.Close()
}

func (g *Generator) GenService() {
	if !g.UseGRPC && !g.UseHTTP {
		return
	}
	os.MkdirAll(g.Project+"/service", 0755)
	filename := fmt.Sprintf("%s/service/service.go", g.Project)
	g.generate(g.serviceTmpl, filename)
}

func (g *Generator) GenMain() {
	filename := fmt.Sprintf("%s/main.go", g.Project)
	g.generate(g.mainTmpl, filename)
}

func (g *Generator) GenClient() {
	os.MkdirAll(g.Project+"/test", 0755)
	filename := fmt.Sprintf("%s/test/main.go", g.Project)
	g.generate(g.clientTmpl, filename)
}

func (g *Generator) generate(tmpl *template.Template, filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	err = tmpl.Execute(f, *g)
	if err != nil {
		log.Panic(err)
	}
	f.Sync()
	f.Close()
}

func (g *Generator) RunOptionList(generator Generator) string {
	s := "// run with master-worker\n"
	s += "    mw.Run("
	if generator.UseGRPC {
		s += "mw.WithServer(grpcServer), "
	}
	if generator.UseHTTP {
		s += "mw.WithServer(httpServer), "
	}
	if generator.UseGW {
		s += "mw.WithServer(gwServer), "
	}
	if generator.UseCC {
		s += "mw.WithConfCenter(cc), "
	}
	s = strings.TrimSuffix(s, ", ")
	s += ")"
	return s
}

func main() {
	app := cli.NewApp()
	app.Name = "box-cli"
	app.Usage = "The command line tool for the microservice framework box"
	app.Version = "0.2.0"
	app.Commands = []cli.Command{
		{
			Name:      "new",
			Usage:     "create a new project",
			Action:    newSkeleton,
			UsageText: "box-cli new project [arguments...]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "confcenter",
					Usage: "use config center",
				},
				cli.BoolFlag{
					Name:  "grpc",
					Usage: "Server based on GRPC protocol",
				},
				cli.BoolFlag{
					Name:  "http",
					Usage: "Server based on HTTP protocol",
				},
				cli.BoolFlag{
					Name:  "gateway",
					Usage: "Convert the HTTP protocol to the GRPC protocol",
				},
				cli.BoolFlag{
					Name:  "registry",
					Usage: "use registry center",
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func newSkeleton(ctx *cli.Context) {
	project := ctx.Args().First()
	defer func() {
		if err := recover(); err != nil {
			//os.RemoveAll(project)
			log.Error(err)
		}
	}()

	g := NewGenerator()
	g.Project = project
	g.CCProject = CamelCase(project)
	g.UseGRPC = ctx.Bool("grpc")
	g.UseHTTP = ctx.Bool("http")
	g.UseGW = ctx.Bool("gateway")
	g.UseRegistry = ctx.Bool("registry")
	g.UseCC = ctx.Bool("confcenter")
	pkgDir, goPath := getPkgDir()
	g.GoPath = goPath
	g.PkgPath = pkgDir + "/" + g.Project

	g.GenConf()
	g.GenPB()
	g.GenService()
	g.GenMain()
	g.GenClient()
}

const serviceCode = `package service

import (
    "context"

    {{- if .UseGRPC}}
    pb "{{.PkgPath}}/pb"
    {{- end}}
)

type service struct{}

func NewService() *service {
    return &service{}
}

func (h *service) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoReply, error) {
    return &pb.EchoReply{Msg: in.Msg}, nil
}
`

const pbCode = `syntax = "proto3";

package {{.Project}};

import "google/api/annotations.proto";

service {{.CCProject}} {
  rpc Echo (EchoRequest) returns (EchoReply) {
  		{{- if or .UseHTTP .UseGW}}
        option (google.api.http) = {
            post: "/echo"
            body: "*"
        };
  		{{- end}}
  }
}

message EchoRequest {
  string msg = 1;
}

message EchoReply {
  string msg = 1;
}
`

const confCode = `package conf

import (
    "io/ioutil"

    yaml "gopkg.in/yaml.v2"
)

type Conf struct {` + "\n" +
	"    LogPath  string `" + `yaml:"logpath"` + "`\n" +
	"    LogLevel int    `" + `yaml:"loglevel"` + "`" +
	`
}

func NewConf(path string, conf *Conf) error {
    file, err := ioutil.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(file, conf)
}
`

const mainCode = `// Code generated by box-cli. YOU CAN EDIT.

package main

import (
    "flag"
	{{- if .UseGW}}
	"context"
	{{- end}}
    {{- if or .UseCC .UseRegistry}}
    "strings"
    "time"

    etcd "github.com/coreos/etcd/clientv3"
    {{- end}}
	{{- if .UseGW}}
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	{{- end}}

    {{- if or .UseCC}}
    "github.com/tddhit/box/confcenter"
    {{- end}}
    "github.com/tddhit/box/mw"
    {{- if .UseRegistry}}
    "github.com/tddhit/box/naming"
    tropt "github.com/tddhit/box/transport/option"
    {{- end}}
    {{- if or .UseGRPC .UseHTTP .UseGW}}
    "github.com/tddhit/box/transport"
    {{- end}}
    "github.com/tddhit/tools/log"
    . "{{.PkgPath}}/conf"
    {{- if or .UseGRPC .UseHTTP}}
    "{{.PkgPath}}/service"
    {{- end}}
    {{- if or .UseGRPC .UseHTTP .UseGW}}
    pb "{{.PkgPath}}/pb"
    {{- end}}
	{{- if .UseGW}}
	tropt "github.com/tddhit/box/transport/option"
	{{- end}}
)

var (
    {{- if or .UseCC .UseRegistry}}
    etcdAddrs  string
    {{- end}}
    {{- if .UseCC}}
    confKey    string
    {{- end}}
    confPath   string
    {{- if .UseGRPC}}
    grpcAddr string
    {{- end}}
    {{- if .UseHTTP}}
    httpAddr string
    {{- end}}
    {{- if .UseGW}}
    gwAddr string
	gwBackend string
    {{- end}}
    {{- if and .UseRegistry .UseGRPC}}
    grpcRegistry   string
    {{- end}}
    {{- if and .UseRegistry .UseHTTP}}
    httpRegistry   string
    {{- end}}
)

func init() {
    {{- if or .UseCC .UseRegistry}}
    flag.StringVar(&etcdAddrs, "etcd-addrs", "127.0.0.1:2379", "etcd endpoints")
    {{- end}}
    {{- if .UseCC}}
    flag.StringVar(&confKey, "conf-key", "/config/{{.Project}}", "config key")
    {{- end}}
    flag.StringVar(&confPath, "conf-path", "conf/{{.Project}}.yml", "config file")
    {{- if and .UseRegistry .UseGRPC}}
    flag.StringVar(&grpcRegistry, "grpc-registry", "/service/grpc/{{.Project}}", "grpc registry target")
    {{- end}}
    {{- if and .UseRegistry .UseHTTP}}
    flag.StringVar(&httpRegistry, "http-registry", "/service/http/{{.Project}}", "http registry target")
    {{- end}}
    {{- if .UseGRPC}}
	flag.StringVar(&grpcAddr, "grpc-addr", "grpc://127.0.0.1:9000", "grpc listen address")
    {{- end}}
    {{- if .UseHTTP}}
	flag.StringVar(&httpAddr, "http-addr", "http://127.0.0.1:9010", "http listen address")
    {{- end}}
    {{- if .UseGW}}
	flag.StringVar(&gwAddr, "gateway-addr", "http://127.0.0.1:9020", "gateway listen address")
	flag.StringVar(&gwBackend, "gateway-backend", "127.0.0.1:9000", "grpc backend server address")
    {{- end}}
    flag.Parse()
}

func main() {
    var (
		conf  	Conf
		err 	error
	)
    {{- if or .UseCC .UseRegistry}}
    // new etcd client
    endpoints := strings.Split(etcdAddrs, ",")
    ec, err := etcd.New(
        etcd.Config{
            Endpoints:   endpoints,
            DialTimeout: time.Second,
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    {{- end}}

    {{- if not .UseCC}}
    NewConf(confPath, &conf)
	{{- else}}
    // new conf center
    cc := confcenter.New(ec, confKey, confcenter.WithSavePath(confPath))
    cc.MakeConf(&conf)
    {{- end}}

    {{- if .UseRegistry}}
    r := naming.NewRegistry(ec)
    {{- end}}
    log.Init(conf.LogPath, conf.LogLevel)
    {{- if or .UseGRPC .UseHTTP}}
    service := service.NewService()
	{{- end}}

    {{if .UseGRPC}}
    // new GRPC Server
	{{- if .UseRegistry}}
    grpcServer, err := transport.Listen(grpcAddr, tropt.WithRegistry(r, grpcRegistry))
	{{- else}}
    grpcServer, err := transport.Listen(grpcAddr)
    {{- end}}
    if err != nil {
        log.Fatal(err)
    }
    grpcServer.Register(pb.{{.CCProject}}GrpcServiceDesc, service)
    {{end}}


	{{if .UseHTTP}}
    // new HTTP Server
	{{- if .UseRegistry}}
    httpServer, err := transport.Listen(httpAddr, tropt.WithRegistry(r, httpRegistry))
	{{- else}}
    httpServer, err := transport.Listen(httpAddr)
	{{- end}}
    if err != nil {
        log.Fatal(err)
    }
    httpServer.Register(pb.{{.CCProject}}HttpServiceDesc, service)
    {{end}}

	{{if .UseGW}}
    // new Gateway Server
	mux := runtime.NewServeMux()
	err = pb.Register{{.CCProject}}HandlerFromEndpoint(context.Background(),
		mux, gwBackend, []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		log.Fatal(err)
	}
    gwServer, err := transport.Listen(gwAddr, tropt.WithGatewayMux(mux))
    if err != nil {
        log.Fatal(err)
    }
    {{end}}

    {{RunOptionList .}}
}
`

const clientCode = `package main

import (
	"context"
	"time"

    pb "{{.PkgPath}}/pb"
	"github.com/tddhit/box/transport"
	"github.com/tddhit/tools/log"
)

func main() {
	{{- if .UseGRPC}}
	{
		conn, err := transport.Dial("grpc://127.0.0.1:9000")
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		c := pb.New{{.CCProject}}GrpcClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reply, err := c.Echo(ctx, &pb.EchoRequest{Msg: "hello"})
		if err != nil {
			log.Fatalf("could not echo: %v", err)
		}
		log.Debug("Grpc Echo: ", reply.Msg)
	}
	{{- end}}

	{{- if .UseHTTP}}
	{
			conn, err := transport.Dial("http://127.0.0.1:9010")
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		c := pb.New{{.CCProject}}HttpClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 
				time.Second)
		defer cancel()
		reply, err := c.Echo(ctx, &pb.EchoRequest{Msg: "hello"})
		if err != nil {
			log.Fatalf("could not echo: %v", err)
		}
		log.Debug("Http Echo: ", reply.Msg)
	}
	{{- end}}

	{{- if .UseGW}}
	{
		conn, err := transport.Dial("http://127.0.0.1:9020")
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		c := pb.New{{.CCProject}}HttpClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 
				time.Second)
		defer cancel()
		reply, err := c.Echo(ctx, &pb.EchoRequest{Msg: "hello"})
		if err != nil {
			log.Fatalf("could not echo: %v", err)
		}
		log.Debug("Gateway Echo: ", reply.Msg)
	}
	{{- end}}
}
`
