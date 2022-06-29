package pl

import (
	"bytes"

	// go template
	"text/template"

	// pongo
	"github.com/flosch/pongo2"

	// markdown
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Template interface {
	Compile(name, input string, opt Val) error
	Execute(context Val) (string, error)
}

type TemplateFactory interface {
	Create() Template
}

type goTemplate struct {
	goT *template.Template
}

func (t *goTemplate) Compile(name, input string, _ Val) error {
	tp, err := template.New(name).Parse(input)
	if err != nil {
		return err
	}
	t.goT = tp
	return nil
}

func (t *goTemplate) Execute(ctx Val) (string, error) {
	x := new(bytes.Buffer)
	err := t.goT.Execute(x, ctx.ToNative())
	if err != nil {
		return "", err
	}
	return x.String(), nil
}

// for now markdown is static at all, ie no runtime rendering what's so ever
type mdTemplate struct {
	md string
}

func (t *mdTemplate) Compile(_, input string, _ Val) error {
	r := html.NewRenderer(
		html.RendererOptions{Flags: html.CommonFlags})

	txt := markdown.ToHTML([]byte(input), nil, r)
	t.md = string(txt)
	return nil
}

func (t *mdTemplate) Execute(ctx Val) (string, error) {
	return t.md, nil
}

type pongoTemplate struct {
	tpl *pongo2.Template
}

func (t *pongoTemplate) Compile(_, input string, _ Val) error {
	r, err := pongo2.FromString(input)
	if err != nil {
		return err
	}
	t.tpl = r
	return nil
}

func (t *pongoTemplate) tocontext(v Val) pongo2.Context {
	switch v.Type {
	case ValPair:
		return pongo2.Context{
			"first":  v.Pair().First.ToNative(),
			"second": v.Pair().Second.ToNative(),
		}

	case ValMap:
		p := make(pongo2.Context)
		v.Map().Foreach(
			func(k string, v Val) bool {
				p[k] = v.ToNative()
				return true
			},
		)
		return p

	default:
		return make(pongo2.Context)
	}
}

func (t *pongoTemplate) Execute(ctx Val) (string, error) {
	return t.tpl.Execute(t.tocontext(ctx))
}

type gotempfac struct{}

func (f *gotempfac) Create() Template {
	return &goTemplate{}
}

type mdtempfac struct{}

func (f *mdtempfac) Create() Template {
	return &mdTemplate{}
}

type pongotempfac struct{}

func (f *pongotempfac) Create() Template {
	return &pongoTemplate{}
}

// Public interface to allow user to register multiple different template engine
// into PL language environment for customization
var templatefacmap = make(map[string]TemplateFactory)

func AddTemplateFactory(name string, f TemplateFactory) {
	templatefacmap[name] = f
}

func init() {
	AddTemplateFactory("go", &gotempfac{})
	AddTemplateFactory("md", &mdtempfac{})
	AddTemplateFactory("pongo", &pongotempfac{})
}

func newTemplate(t string) Template {
	f, ok := templatefacmap[t]
	if ok {
		return f.Create()
	} else {
		return nil
	}
}
