package hpl

import (
	"bytes"

	// go template
	"text/template"

	// markdown
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Template interface {
	Compile(name, input string, opt Val) error
	Execute(context Val) (string, error)
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

func newTemplate(t string) Template {
	switch t {
	case "go":
		return &goTemplate{}
	case "md":
		return &mdTemplate{}
	default:
		return nil
	}
}
