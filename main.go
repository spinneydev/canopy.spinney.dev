// Command site builds canopy.spinney.dev: the pages in content/, and the
// documentation from the markdown in the canopy folder, all set in the
// templates in templates/. content/home.md becomes public/index.html, every
// other page, such as content/clone.md, public/clone/index.html, and each
// document public/docs/<slug>/index.html. Run it from the site folder:
//
//	go run .            # regenerate the pages in public/
//	go run . -serve     # the same, then serve public/ on 127.0.0.1:8000,
//	                    # rebuilding on every page load
//	go run . -src ../canopy -serve -addr 127.0.0.1:9000
//
// With no canopy folder at -src, as on Cloudflare Pages, which checks out
// only the site, it clones canopy from GitHub into it first.
//
// A page is a markdown file with its title, and optionally a description,
// between two --- lines on top:
//
//	---
//	title: Clone
//	description: Clone canopy and try it with sample repositories.
//	---
//
//	## 1. Get the source
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// canopyURL is where canopy is cloned from when there is no copy at -src.
const canopyURL = "https://github.com/spinneydev/canopy"

// sourceURL is where links to files other than the documentation go.
const sourceURL = "https://github.com/spinneydev/canopy/blob/main/"

type doc struct {
	Slug  string // the page is docs/<slug>/
	Src   string // markdown file, relative to the canopy folder
	Title string // short name for navigation
	Blurb string
}

// docs is in reading order: the order someone setting up a server needs
// them. The sidebar and the previous/next links follow it.
var docs = []doc{
	{"deploy", "deploy.md", "Deploying", "Putting it on a server, from a fresh VPS to a hardened setup with backups."},
	{"usage", "usage.md", "Using", "Day to day: cloning, pushing, pulling, new repositories, a short SSH name for the server."},
	{"configuration", "config/README.md", "Configuration", "Every setting in config/canopy.toml."},
	{"customisation", "customisation.md", "Customising", "Themes, your own colours and fonts, step by step."},
	{"security", "security.md", "Security", "How the server is protected, layer by layer."},
	{"development", "development.md", "Development", "Running and testing it on your own machine, including Windows."},
}

type heading struct {
	ID, Text string
}

// part is one document, rendered.
type part struct {
	doc
	Heading string
	TOC     []heading
	Body    template.HTML
}

// page is what the templates get. The docs fields are set only on the
// documentation's pages, and Doc only on a document's own page.
type page struct {
	Name        string // the content file's name without .md, or "docs"
	Title       string
	Description string
	Body        template.HTML

	Doc        *part
	Docs       []part
	Prev, Next *part
}

func main() {
	src := flag.String("src", "../canopy", "the canopy folder")
	out := flag.String("out", "public", "the site folder")
	serve := flag.Bool("serve", false, "serve the site after building it, and rebuild on every page load")
	addr := flag.String("addr", "127.0.0.1:8000", "address to serve on, with -serve")
	flag.Parse()
	log.SetFlags(0)

	if err := fetch(*src); err != nil {
		log.Fatal(err)
	}
	if err := build(*src, *out); err != nil {
		log.Fatal(err)
	}
	if !*serve {
		return
	}
	// One build at a time: a build clears docs/ before writing it again.
	var mu sync.Mutex
	files := http.FileServer(http.Dir(*out))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; strings.HasSuffix(p, "/") || strings.HasSuffix(p, ".html") {
			mu.Lock()
			err := build(*src, *out)
			mu.Unlock()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
	log.Printf("serving %s on http://%s/", *out, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// fetch clones canopy into src if nothing is there. A folder that is
// there is used as it is, never updated: keeping it current is up to
// whoever made it.
func fetch(src string) error {
	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	log.Printf("no canopy at %s: cloning %s", src, canopyURL)
	cmd := exec.Command("git", "clone", "--quiet", "--depth", "1", canopyURL, src)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloning canopy: %w", err)
	}
	return nil
}

// build writes a page into out for every markdown file in content/, the
// documentation into out/docs/, and out/chroma.css.
func build(src, out string) error {
	t, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return err
	}

	pages, err := filepath.Glob("content/*.md")
	if err != nil {
		return err
	}
	for _, name := range pages {
		p, err := parse(name)
		if err != nil {
			return err
		}
		dir := p.Name
		if dir == "home" {
			dir = ""
		}
		if err := write(t, filepath.Join(out, dir, "index.html"), "layout", p); err != nil {
			return err
		}
	}

	// Every page's sidebar lists every document, so convert them all
	// before writing any.
	bySource := map[string]string{}
	for _, d := range docs {
		bySource[d.Src] = d.Slug
	}
	var parts []part
	for _, d := range docs {
		md, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(d.Src)))
		if err != nil {
			return err
		}
		p := part{doc: d}
		if p.Heading, p.TOC, p.Body, err = convert(md, d, bySource); err != nil {
			return fmt.Errorf("%s: %w", d.Src, err)
		}
		parts = append(parts, p)
	}

	// Only now, with every document read, replace docs/: a failed read
	// leaves the last good pages in place, and pages of documents that
	// are gone go with the rest.
	if err := os.RemoveAll(filepath.Join(out, "docs")); err != nil {
		return err
	}
	welcome := page{
		Name:        "docs",
		Title:       "Documentation",
		Description: "Everything about running your own canopy server.",
		Docs:        parts,
		Next:        &parts[0],
	}
	if err := write(t, filepath.Join(out, "docs", "index.html"), "welcome", welcome); err != nil {
		return err
	}
	for i := range parts {
		p := page{
			Name:        "docs",
			Title:       parts[i].Title,
			Description: parts[i].Blurb,
			Doc:         &parts[i],
			Docs:        parts,
		}
		if i > 0 {
			p.Prev = &parts[i-1]
		}
		if i < len(parts)-1 {
			p.Next = &parts[i+1]
		}
		if err := write(t, filepath.Join(out, "docs", parts[i].Slug, "index.html"), "doc", p); err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(out, "chroma.css"), highlightCSS("github", "github-dark"), 0o644)
}

func write(t *template.Template, name, tmpl string, p page) error {
	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, tmpl, p); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, b.Bytes(), 0o644)
}

// markdown is the converter for every page: GitHub's flavour, ids on
// headings, highlighted code, and HTML let through for the pages in
// content/. The documents add a transformer of their own.
func markdown(opts ...parser.Option) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(append(opts, parser.WithAutoHeadingID())...),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(codeBlockRenderer{}, 200)),
		),
	)
}

// parse reads one page from content/: its front matter, then its body.
func parse(src string) (page, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return page{}, err
	}
	p := page{Name: strings.TrimSuffix(filepath.Base(src), ".md")}
	parts := strings.SplitN(strings.ReplaceAll(string(raw), "\r\n", "\n"), "---\n", 3)
	if len(parts) != 3 || parts[0] != "" {
		return page{}, fmt.Errorf("%s: no front matter: start with ---, title: ..., ---", src)
	}
	for _, line := range strings.Split(parts[1], "\n") {
		k, v, _ := strings.Cut(line, ":")
		switch strings.TrimSpace(k) {
		case "title":
			p.Title = strings.TrimSpace(v)
		case "description":
			p.Description = strings.TrimSpace(v)
		}
	}
	if p.Title == "" {
		return page{}, fmt.Errorf("%s: no title in the front matter", src)
	}
	var b bytes.Buffer
	if err := markdown().Convert([]byte(parts[2]), &b); err != nil {
		return page{}, fmt.Errorf("%s: %w", src, err)
	}
	p.Body = template.HTML(b.String())
	return p, nil
}

// convert renders one document. It returns the text of its top heading
// (which is left out of the body), its second-level headings for the table
// of contents, and the body.
func convert(src []byte, d doc, bySource map[string]string) (string, []heading, template.HTML, error) {
	c := &collector{dir: path.Dir(d.Src), bySource: bySource}
	var b bytes.Buffer
	if err := markdown(parser.WithASTTransformers(util.Prioritized(c, 100))).Convert(src, &b); err != nil {
		return "", nil, "", err
	}
	return c.title, c.toc, template.HTML(b.String()), nil
}

// collector rewrites links between documents, picks out the headings and
// removes the top heading, which the page shows in its own header.
type collector struct {
	dir      string
	bySource map[string]string // markdown file to slug
	title    string
	toc      []heading
}

func (c *collector) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	src := reader.Source()
	var drop ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Heading:
			switch {
			case n.Level == 1 && c.title == "":
				c.title = plain(n, src)
				drop = n
			case n.Level == 2:
				id, _ := n.AttributeString("id")
				c.toc = append(c.toc, heading{string(id.([]byte)), plain(n, src)})
			}
		case *ast.Link:
			n.Destination = c.rewrite(n.Destination)
		}
		return ast.WalkContinue, nil
	})
	if drop != nil {
		drop.Parent().RemoveChild(drop.Parent(), drop)
	}
}

// rewrite points links to other documents at their pages, and links to
// any other file in the repository at its source on GitHub.
func (c *collector) rewrite(dest []byte) []byte {
	u, err := url.Parse(string(dest))
	if err != nil || u.Scheme != "" || u.Host != "" || u.Path == "" {
		return dest
	}
	p := strings.TrimPrefix(path.Clean(path.Join(c.dir, u.Path)), "/")
	if slug, ok := c.bySource[p]; ok {
		return []byte("/docs/" + slug + "/" + fragment(u))
	}
	return []byte(sourceURL + p + fragment(u))
}

func fragment(u *url.URL) string {
	if u.Fragment == "" {
		return ""
	}
	return "#" + u.Fragment
}

func plain(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// codeBlockRenderer highlights fenced code blocks by their info string, as
// canopy does for READMEs. Blocks without a known language stay plain.
type codeBlockRenderer struct{}

func (codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, renderCodeBlock)
	reg.Register(ast.KindCodeBlock, renderCodeBlock)
}

var formatter = chromahtml.New(chromahtml.WithClasses(true))

func renderCodeBlock(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	var code bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		code.Write(seg.Value(src))
	}
	var lexer chroma.Lexer
	if f, ok := n.(*ast.FencedCodeBlock); ok {
		if lang := string(f.Language(src)); lang != "" {
			lexer = lexers.Get(lang)
		}
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, code.String())
	if err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, formatter.Format(w, styles.Fallback, it)
}

// highlightCSS is the stylesheet for highlighted code: the light style
// outside dark mode and the dark style inside it.
func highlightCSS(light, dark string) []byte {
	var b bytes.Buffer
	b.WriteString("/* Generated by main.go. */\n")
	b.WriteString("@media not all and (prefers-color-scheme: dark) {\n")
	formatter.WriteCSS(&b, styles.Get(light))
	b.WriteString("}\n\n@media (prefers-color-scheme: dark) {\n")
	formatter.WriteCSS(&b, styles.Get(dark))
	b.WriteString("}\n")
	// The page provides its own code background: site.css's .prose pre
	// outranks .chroma.
	b.WriteString(".chroma .bg { background: transparent; }\n")
	return b.Bytes()
}
