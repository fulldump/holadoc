package main

import (
	"bytes"
	"cmp"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma"
	html2 "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/fulldump/goconfig"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkHtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type config struct {
	Src       string `json:"src"`
	Www       string `json:"www"`
	Versions  string `json:"versions" usage:"default version is the first one"`
	Languages string `json:"languages" usage:"default language is the first one"`
	Serve     string `json:"serve" usage:"Address to serve files locally, example ':8080'"`
}

// todo: avoid globals
var versions []string
var languages []string

func main() {

	c := config{
		Src:       "src/",
		Www:       "www/",
		Versions:  "v1,v2",
		Languages: "en,es,zh",
	}
	goconfig.Read(&c)

	if c.Serve != "" {

		s := &http.Server{
			Addr:    c.Serve,
			Handler: http.FileServer(http.Dir(c.Www)),
		}

		s.ListenAndServe()
		return
	}

	// clear output
	_ = os.RemoveAll(c.Www)
	err := os.MkdirAll(c.Www, 0777)
	if err != nil {
		panic(err.Error())
	}

	versions = strings.Split(c.Versions, ",")
	languages = strings.Split(c.Languages, ",")

	root := &Node{}

	readNodes(root, c.Src, c.Www)
	root.PrettyPrint(0)

	traverseNodes(root, func(node *Node) {

		for _, version := range versions {
			for _, language := range languages {

				variation := getBestVariation(node.Variations, language, version)
				if variation == nil {
					fmt.Println("skip:", node.Path)
					continue
				}

				langMenu := ""
				{
					langMenu += `<div class="languages">`
					for _, l := range languages {
						class := ""
						if l == language {
							class += "selected"
						}
						langMenu += `<a class="` + class + `" href="` + getLink(node, l, version) + `">` + l + `</a>`
					}
					langMenu += `</div>`
				}

				versionMenu := ""
				{
					if hasVersions(node) {
						versionMenu += `<div class="versions">`
						for _, v := range versions {
							class := ""
							if v == version {
								class += "selected"
							}
							versionMenu += `<a class="` + class + `" href="` + getLink(node, language, v) + `">` + v + `</a>`
						}
						versionMenu += `</div>`
					}
				}

				onThisPage := ""

				content := ""

				{ // content

					var htmlReader io.Reader

					switch strings.ToLower(path.Ext(variation.Filename)) {
					case ".html":
						src, err := os.Open(variation.Filename)
						if err != nil {
							panic(err.Error())
						}
						htmlReader = src
					case ".md":
						src, err := os.ReadFile(variation.Filename)
						if err != nil {
							panic(err.Error())
						}
						htmlReader = md2html(src)
						// htmlReader = strings.NewReader("Hello, this is the new md!")
					}

					doc := &html.Node{
						Type:     html.ElementNode,
						Data:     "body",
						DataAtom: atom.Body,
					}
					nodes, err := html.ParseFragment(htmlReader, doc)
					if err != nil {
						panic(err.Error())
					}

					{ // index

						for _, n := range nodes {
							traverseHtml(n, func(node *html.Node) {
								tag := strings.ToLower(node.Data)
								if in([]string{"h2", "h3", "h4", "h5", "h6"}, tag) && node.FirstChild != nil {
									title := node.FirstChild.Data
									node.Attr = append(node.Attr, html.Attribute{
										Key: "id",
										Val: url.PathEscape(title), // todo: slug?
									})
									onThisPage += `<div class="index-` + node.Data + `">` + "\n"
									onThisPage += `<a href="#` + url.PathEscape(title) + `">` + title + `</a>` + "\n"
									onThisPage += `</div>` + "\n"
								}
								if tag == "a" {
									href := getAttribute(node, "href")
									if href != "" {
										target := getNode(root, href)
										if target != nil {
											setAttribute(node, "href", getLink(target, variation.Language, variation.Version))
											if node.FirstChild != nil && node.FirstChild.FirstChild == nil && node.FirstChild.Type == html.TextNode {
												node.FirstChild.Data = target.Name
											} else if node.FirstChild == nil {
												node.AppendChild(&html.Node{
													Type:     html.TextNode,
													DataAtom: 0,
													Data:     target.Name,
												})
											}
										}
									}
								}
								if tag == "code" {
									code := node.FirstChild.Data
									code = strings.TrimPrefix(code, "\n")

									lexer := lexers.Get(getAttribute(node, "lang"))
									if lexer == nil {
										lexer = lexers.Get(strings.TrimPrefix(getAttribute(node, "class"), "language-"))
									}
									if lexer == nil {
										lexer = lexers.Analyse(code)
									}
									if lexer == nil {
										lexer = lexers.Fallback
									}
									lexer = chroma.Coalesce(lexer)

									style := styles.Get("solarized-dark") // monokai github-dark
									if style == nil {
										style = styles.Fallback
									}
									formatter := html2.New(html2.WithLineNumbers(true), html2.LinkableLineNumbers(true, "L"))

									iterator, err := lexer.Tokenise(nil, code)

									codeOutput := &bytes.Buffer{}
									err = formatter.Format(codeOutput, style, iterator)
									if err != nil {
										panic(err.Error())
									}

									node.RemoveChild(node.FirstChild)

									doc := &html.Node{
										Type:     html.ElementNode,
										Data:     "body",
										DataAtom: atom.Body,
									}

									parts, err := html.ParseFragment(codeOutput, doc)
									if err != nil {
										panic(err)
									}
									for _, part := range parts {
										node.AppendChild(part)
									}

								}
							})
						}
					}

					{ // print content
						b := &bytes.Buffer{}

						// if variation.Version != "" && version > variation.Version { // todo: make this comparison better (taking into account numbers, not only strings)
						// 	fmt.Fprintln(b, `<div class="alert">This has been unchanged since version `+variation.Version+`</div>`)
						// }

						for _, n := range nodes {
							html.Render(b, n)
						}
						content = b.String()
					}

				}

				data := map[string]any{
					"lang":        variation.Language,
					"langs":       languages,
					"langMenu":    template.HTML(langMenu),
					"title":       variation.Title,
					"url":         variation.Url,
					"filename":    variation.Filename,
					"version":     variation.Version,
					"versions":    versions,
					"versionMenu": template.HTML(versionMenu),
					"tree":        template.HTML(getIndex(root, node, language, version)),
					"breadcrumb":  template.HTML(getBreadcrumb(node, language, version)),
					"index":       template.HTML(onThisPage),
					"content":     template.HTML(content),
				}

				newFilename := path.Join(c.Www, getOutputPath(node, variation, language, version))
				os.MkdirAll(path.Dir(newFilename), 0777) // todo: handle err

				f, err := os.Create(newFilename)
				if err != nil {
					panic(err.Error())
				}

				temp := getTemplate(node, map[string]any{
					"link": func(p string) template.HTML {

						target := getNode(root, p)
						if target == nil {
							panic("link for '" + p + "' does not exist")
						}

						class := "link"
						if target == node {
							class += " selected"
						}

						variation := getBestVariation(target.Variations, language, version)

						return template.HTML(`<a class="` + class + `" href="` + getLink(target, language, version) + `">` + variation.Title + `</a>`)
					},

					"tree": func(p string) template.HTML {

						target := getNode(root, p)
						if target == nil {
							panic("link for '" + p + "' does not exist")
						}

						return template.HTML(getIndex(target, node, language, version))
					},

					"isUnder": func(p string) bool {

						target := getNode(root, p)
						if target == nil {
							panic("link for '" + p + "' does not exist")
						}

						n := node

						for n != nil {
							if n == target {
								return true
							}
							n = n.Parent
						}

						return false
					},
				})
				temp.Execute(f, data)

				err = f.Close()
				if err != nil {
					panic(err.Error())
				}

			}
		}

	})

}

func getTemplate(node *Node, funcs template.FuncMap) *template.Template {

	for node != nil {
		if node.Template == "" {
			node = node.Parent
			continue
		}

		gohtml, err := os.ReadFile(node.Template)
		if err != nil {
			panic(err.Error())
		}

		temp, err := template.New("").Funcs(funcs).Parse(string(gohtml))
		if err != nil {
			panic(err.Error())
		}

		return temp
	}

	panic("No template found!!!")

	return nil
}

func getNode(root *Node, path string) *Node {
	if path == "" {
		return root
	}

	n := root
out:
	for _, p := range strings.Split(path, "/") {
		for _, child := range n.Children {
			if child.Name == p {
				n = child
				continue out
			}
		}
		return nil
	}

	return n
}

func getAttribute(node *html.Node, key string) string {
	for _, a := range node.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttribute(node *html.Node, key, value string) {
	for i, a := range node.Attr {
		if strings.EqualFold(a.Key, key) {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{
		Key: key,
		Val: value,
	})
}

func hasVersions(node *Node) bool {
	if node == nil {
		return false
	}

	if node.Name == "{version}" {
		return true
	}

	return hasVersions(node.Parent)
}

// return the best variation for a given language and version
func getBestVariation(variations []*Variation, language, version string) *Variation {
	var variation *Variation

	// choose best possible variation

	for _, v := range variations {
		if v.Language == language && v.Version == version {
			variation = v
			break
		}
	}

	// fallback by version
	if variation == nil {
		// todo: sort and filter by version (should be less or eq than "version")
		for _, v := range variations {
			if v.Version == version {
				variation = v
				break
			}
		}
	}

	// fallback by language
	if variation == nil {
		for _, v := range variations {
			if v.Language == language {
				variation = v
				break
			}
		}
	}

	// fallback by any variation (version must be less or equal!)
	if variation == nil {
		for _, v := range variations {
			variation = v
			break
		}
	}

	return variation
}

var basepath = "/"

func getLink(n *Node, lang, version string) string {
	variation := getBestVariation(n.Variations, lang, version)
	return path.Join(basepath, getOutputPath(n, variation, lang, version))
}

func getBreadcrumb(n *Node, lang, version string) string {
	breadcrumb := []*Node{}

	for n != nil && len(n.Variations) > 0 {
		if n.Parent == nil {
			break
		}
		breadcrumb = append(breadcrumb, n)
		n = n.Parent
	}

	if len(breadcrumb) < 2 {
		return ""
	}

	slices.Reverse(breadcrumb)

	result := ""

	result += `<div class="breadcrumb">`
	for i, node := range breadcrumb {
		if node.Name == "{version}" {
			continue
		}
		if i > 0 {
			result += `<span class="arrow">→</span>`
		}
		v := getBestVariation(node.Variations, lang, version)
		class := "item"
		if i == len(breadcrumb)-1 {
			class += " selected"
		}
		result += `<a class="` + class + `" href="` + getLink(node, lang, version) + `">` + v.Title + `</a>`
	}
	result += `</div>`

	return result
}

func getIndex(root, target *Node, lang, version string) string {

	nodesToParent := []*Node{}
	n := target
	for n != nil {
		nodesToParent = append(nodesToParent, n)
		n = n.Parent
	}

	result := ""

	for _, child := range root.Children {

		if child.Name == "{version}" {
			result += getIndex(child, target, lang, version)
			continue
		}

		link := getLink(child, lang, version)

		variation := getBestVariation(child.Variations, lang, version)

		class := "item"
		if nodeIn(nodesToParent, child) {
			class += " active"
		}
		if child == target {
			class += " selected"
		}

		result += `<div class="` + class + `"><a href="` + link + `">` + variation.Title + `</a></div>` + "\n"

		if len(child.Children) == 0 {
			continue
		}

		result += `<div class="children">` + "\n"
		result += getIndex(child, target, lang, version)
		result += `</div>` + "\n"
	}

	return result
}

func traverseNodes(root *Node, callback func(*Node)) {

	callback(root)
	for _, child := range root.Children {
		// callback(child)
		traverseNodes(child, callback)
	}
}

type Node struct {
	Order      int
	Name       string
	Path       string
	Children   []*Node
	Variations []*Variation
	Parent     *Node
	Template   string
}

type Variation struct {
	Url      string
	Language string
	Version  string
	Filename string
	Title    string
}

func (n *Node) PrettyPrint(indent int) {
	for _, child := range n.Children {
		fmt.Printf("%s (%d)\n", strings.Repeat("    ", indent)+child.Name, len(child.Variations))
		child.PrettyPrint(indent + 1)
	}
}

func getOutputPath(node *Node, variation *Variation, lang, version string) string {

	if variation == nil {
		return "NIL VARIATION!!! panic(?)"
	}

	result := []string{}

	for node != nil && node.Parent != nil {

		p := ""

		for _, v := range node.Variations {
			// todo: take version into account :S
			if v.Language == variation.Language {
				p = v.Url
				break
			}
		}

		if p == "" {
			for _, v := range node.Variations {
				if v.Version == variation.Version {
					p = v.Url
					break
				}
			}
		}

		if p == "" {
			p = node.Name // fallback
		}

		if p == "{version}" {
			p = version
		}

		result = append([]string{p}, result...)

		node = node.Parent
	}

	defaultLanguage := languages[0]
	if lang != defaultLanguage {
		result = append([]string{lang}, result...)
	}

	result = append(result, "index.html")

	return path.Join(result...)
}

func copyFile(src, dst string) {
	info, err := os.Stat(src)
	if err != nil {
		panic(err.Error())
	}

	if info.IsDir() {
		err = os.MkdirAll(dst, 0777)
		if err != nil {
			panic(err.Error())
		}

		entries, err := os.ReadDir(src)
		if err != nil {
			panic(err.Error())
		}

		for _, entry := range entries {
			copyFile(path.Join(src, entry.Name()), path.Join(dst, entry.Name()))
		}
		return
	}

	d, err := os.Create(dst)
	if err != nil {
		panic(err.Error())
	}
	s, err := os.Open(src)
	if err != nil {
		panic(err.Error())
	}
	_, err = io.Copy(d, s)
	if err != nil {
		panic(err.Error())
	}
	d.Close()
	s.Close()
}

func readNodes(root *Node, src, www string) { // todo: return errors instead of miserably panic
	entries, err := os.ReadDir(src)
	if err != nil {
		panic(err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() {
			var order int
			var name string
			if entry.Name() == "{version}" {
				name = entry.Name()
			} else {
				parts := strings.Split(entry.Name(), "_")
				if len(parts) != 2 {
					copyFile(path.Join(src, entry.Name()), path.Join(www, entry.Name()))
					continue
				}
				order, err = strconv.Atoi(parts[0])
				if err != nil {
					continue
				}
				name = parts[1]
			}

			newNode := &Node{
				Order: order,
				Name:  name,
				Path:  src,
			}
			readNodes(newNode, path.Join(src, entry.Name()), www)

			newNode.Parent = root
			root.Children = append(root.Children, newNode)

		} else {
			ext := strings.ToLower(path.Ext(entry.Name()))
			if ext == ".gohtml" {
				root.Template = path.Join(src, entry.Name())
				continue
			}
			if !in([]string{".html", ".md"}, ext) {
				copyFile(path.Join(src, entry.Name()), path.Join(www, entry.Name()))
				continue
			}

			base := strings.ToLower(strings.TrimSuffix(path.Base(entry.Name()), path.Ext(entry.Name())))
			parts := strings.Split(base, "_")

			friendlyUrl := parts[0]
			lang := ""
			version := ""

			for _, p := range parts[1:] {
				p = strings.ToLower(p)
				if in(languages, p) {
					lang = p
				}
				if in(versions, p) {
					version = p
				}
			}
			parts = parts[1:]

			filename := path.Join(src, entry.Name())

			title := getTitle(filename)
			if title == "" {
				title = friendlyUrl // fallback
				base, _ := os.Getwd()
				fmt.Printf("WARNING: %s:1 needs a title <h1>\n", path.Join(base, filename))
			}

			root.Variations = append(root.Variations, &Variation{
				Url:      friendlyUrl,
				Language: lang,
				Version:  version,
				Filename: filename,
				Title:    title,
			})

		}

	}

	sort.Slice(root.Children, func(i, j int) bool {
		return root.Children[i].Order < root.Children[j].Order
	})

}

func getTitle(filename string) string {

	var htmlReader io.Reader
	f, err := os.Open(filename)
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	switch strings.ToLower(path.Ext(filename)) {
	case ".html":

		htmlReader = f

	case ".md":
		b, err := io.ReadAll(f)
		if err != nil {
			panic(err.Error())
		}

		htmlReader = md2html(b)
	}

	title := ""
	doc, err := html.Parse(htmlReader)
	if err != nil {
		panic(err.Error())
	}

	traverseHtml(doc, func(node *html.Node) {
		if node.Data == "h1" && node.FirstChild != nil {
			title = node.FirstChild.Data
		}
	})

	return title
}

func traverseHtml(n *html.Node, callback func(node *html.Node)) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		traverseHtml(c, callback)
	}
	if n.Type == html.ElementNode {
		callback(n)
	}
}

func in[T cmp.Ordered](a []T, v T) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}

func nodeIn(a []*Node, v *Node) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}

func md2html(md []byte) io.Reader {

	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
		// parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			goldmarkHtml.WithXHTML(),
			goldmarkHtml.WithUnsafe(),
		),
	)

	buf := &bytes.Buffer{}
	err := gm.Convert(md, buf)
	if err != nil {
		panic(err.Error())
	}
	return buf
}
