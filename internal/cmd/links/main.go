package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type schema struct {
	Type string `yaml:"type"`
}

type parameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
	Schema   schema `yaml:"schema"`
}

type endpoint struct {
	Parameters []parameter `yaml:"parameters"`
}

type path map[string]endpoint

type spec struct {
	Paths map[string]path `yaml:"paths"`
}

var acronyms = []string{
	"api", "id",
}

func makeName(path, method string) string {
	noParens := regexp.MustCompile(`[{}]`).ReplaceAllString(path, "")
	words := []string{method}
	words = append(words, strings.Split(strings.Trim(noParens, "/"), "/")...)
	for i, word := range words {
		word = strings.ToLower(word)
		if slices.Contains(acronyms, word) {
			words[i] = strings.ToUpper(word)
		} else {
			words[i] = strings.Title(word)
		}
	}

	return strings.Join(words, "")
}

func makeArgs(params []parameter) (string, string, map[string]string) {
	args := []string{}
	names := []string{}
	fstrings := map[string]string{}
	for _, param := range params {
		if param.In != "path" {
			continue
		}
		if !param.Required {
			panic("path param not required")
		}
		args = append(args, fmt.Sprintf("%s %s", param.Name, param.Schema.Type))
		names = append(names, param.Name)
		switch param.Schema.Type {
		case "string":
			fstrings[param.Name] = `%s`
		default:
			panic(fmt.Sprintf("not implemented: type %s", param.Schema.Type))
		}
	}

	return strings.Join(args, ", "), strings.Join(names, ", "), fstrings
}

func main() {
	infile := flag.String("infile", "", "name of the OpenAPI YAML file")
	outfile := flag.String("outfile", "", "name of the output Go file")
	pkgname := flag.String("pkgname", "", "name of the Go package name to generate")
	flag.Parse()

	if *infile == "" {
		panic("missing flag -infile")
	}
	if *outfile == "" {
		panic("missing flag -outfile")
	}
	if *pkgname == "" {
		panic("missing flag -pkgname")
	}

	s := spec{}
	{
		inf, err := os.Open(*infile)
		if err != nil {
			panic(err)
		}
		defer inf.Close()

		err = yaml.NewDecoder(inf).Decode(&s)
		if err != nil {
			panic(err)
		}
	}

	outf, err := os.Create(*outfile)
	if err != nil {
		panic(err)
	}
	defer outf.Close()

	fmt.Fprintln(outf, "package", *pkgname)
	fmt.Fprint(outf, `
		import (
			"fmt"
			"context"
			"net/http"
		)

		type Links struct {
			Base string
		}

type ctxKeyLinks struct{}

func WithLinks(ctx context.Context, links Links) context.Context {
	return context.WithValue(ctx, ctxKeyLinks{}, links)
}

func GetLinks(ctx context.Context) *Links {
	if ret, ok := ctx.Value(ctxKeyLinks{}).(Links); ok {
		return &ret
	}
	return nil
}

func MustGetLinks(ctx context.Context) Links {
	links := GetLinks(ctx)
	if links != nil {
		return *links
	}

	panic("no Links attached to context")
}

func AttachLinks(links Links) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = WithLinks(ctx, links)
			h.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
	return f
}

`)

	for path, methods := range s.Paths {
		for method, endpoint := range methods {
			fname := makeName(path, method)
			args, names, fstrings := makeArgs(endpoint.Parameters)

			pathFstr := regexp.MustCompile("{[a-zA-Z0-9]+}").ReplaceAllStringFunc(path, func(s string) string {
				name := strings.Trim(s, "{}")
				return fstrings[name]
			})
			fstr := `%s` + pathFstr

			fmt.Fprintf(outf, "func (l *Links) %s(%s) string {\n", fname, args)
			fmt.Fprintf(outf, `return fmt.Sprintf("%s", l.Base, %s)`, fstr, names)
			fmt.Fprintf(outf, "\n}\n\n")
		}
	}
}
