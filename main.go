package main

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/fulldump/goconfig"
	//    "golang.org/x/net/html"
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
		Versions:  "v2,v1",
		Languages: "en,es",
	}
	goconfig.Read(&c)

	if c.Serve != "" {

		s := &http.Server{
			Addr:    c.Serve,
			Handler: http.FileServer(http.Dir(c.Www)),
		}

		s.ListenAndServe()
	}

	versions = strings.Split(c.Versions, ",")
	languages = strings.Split(c.Languages, ",")

	root := &Node{}

	readNodes(root, c.Src)
	root.PrettyPrint(0)

	// clear output
	_ = os.RemoveAll(c.Www)
	err := os.MkdirAll(c.Www, 0777)
	if err != nil {
		panic(err.Error())
	}

	traverseNodes(root, func(node *Node) {

		for _, version := range versions {
			for _, language := range languages {

				variation := getBestVariation(node.Variations, language, version)
				if variation == nil {
					fmt.Println("skip:", node.Path)
					continue
				}

				newFilename := path.Join(c.Www, version, language, getOutputPath(node, variation)+".html")

				os.MkdirAll(path.Dir(newFilename), 0777) // todo: handle err

				f, err := os.Create(newFilename)
				if err != nil {
					panic(err.Error())
				}

				f.WriteString(
					fmt.Sprintln(`<!--`, variation.Url, variation.Language, variation.Filename, variation.Version, `-->`),
				)

				fmt.Fprintln(f, `<div class="top">`)
				fmt.Fprintln(f, `Languages:`)
				for _, l := range languages {
					fmt.Fprintln(f, `<a href="`+getLink(node, l, version)+`">`+l+`</a>`)
				}
				fmt.Fprintln(f, `<br>`)
				fmt.Fprintln(f, `Versions:`)
				for _, v := range versions {
					fmt.Fprintln(f, `<a href="`+getLink(node, language, v)+`">`+v+`</a>`)
				}
				fmt.Fprintln(f, `</div>`)

				fmt.Fprintln(f, `<style>
.index {
  float: left;
}

.children {
  padding-left: 16px;
}

.content {
  padding-left: 200px;
}

.alert {
  color: dodgerblue;
  background-color: ;
  border: solid dodgerblue 1px;
  padding: 16px;
  border-radius: 4px;
}

.top {
  border-bottom: solid silver 1px;
  margin-bottom: 16px;
}

</style>`)

				f.WriteString(`<div class="index">` + "\n")
				index := getIndex(root, language, version)
				f.WriteString(index)
				f.WriteString(`</div>` + "\n")

				// Process output (for now, just copy the source file)
				src, err := os.Open(variation.Filename)
				if err != nil {
					panic(err.Error())
				}

				f.WriteString(`<div class="content">` + "\n")

				if variation.Version != "" && version > variation.Version { // todo: make this comparison better (taking into account numbers, not only strings)
					fmt.Fprintln(f, `<div class="alert">This has been unchanged since version `+variation.Version+`</div>`)
				}

				io.Copy(f, src)
				f.WriteString(`</div>` + "\n")

				src.Close()

				err = f.Close()
				if err != nil {
					panic(err.Error())
				}

			}
		}

	})

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
	return path.Join(basepath, version, lang, getOutputPath(n, variation)+".html")
}

func getIndex(n *Node, lang, version string) string {

	result := ""

	for _, child := range n.Children {

		link := getLink(child, lang, version)

		result += `<div class="item"><a href="` + link + `">` + child.Name + `</a></div>` + "\n"

		if len(child.Children) == 0 {
			continue
		}

		result += `<div class="children">` + "\n"
		result += getIndex(child, lang, version)
		result += `</div>` + "\n"
	}

	return result
}

func traverseNodes(root *Node, callback func(*Node)) {

	for _, child := range root.Children {
		callback(child)
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
}

type Variation struct {
	Url      string
	Language string
	Version  string
	Filename string
}

func (n *Node) PrettyPrint(indent int) {
	for _, child := range n.Children {
		fmt.Println(strings.Repeat("    ", indent) + child.Name)
		child.PrettyPrint(indent + 1)
	}
}

func getOutputPath(node *Node, variation *Variation) string {

	if variation == nil {
		return "NIL VARIATION!!! panic(?)"
	}

	result := []string{
		variation.Url,
	}

	node = node.Parent

	for node != nil {

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

		result = append([]string{p}, result...)

		node = node.Parent
	}

	return path.Join(result...)
}

func readNodes(root *Node, src string) { // todo: return errors instead of miserably panic
	entries, err := os.ReadDir(src)
	if err != nil {
		panic(err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() {
			parts := strings.Split(entry.Name(), "_")
			if len(parts) != 2 {
				continue
			}
			order, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}

			newNode := &Node{
				Order: order,
				Name:  parts[1],
				Path:  src,
			}
			readNodes(newNode, path.Join(src, entry.Name()))

			newNode.Parent = root
			root.Children = append(root.Children, newNode)

		} else {
			if strings.ToLower(path.Ext(entry.Name())) != ".html" {
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

			root.Variations = append(root.Variations, &Variation{
				Url:      friendlyUrl,
				Language: lang,
				Version:  version,
				Filename: path.Join(src, entry.Name()),
			})

		}

	}

	sort.Slice(root.Children, func(i, j int) bool {
		return root.Children[i].Order < root.Children[j].Order
	})

}

func in[T cmp.Ordered](a []T, v T) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}
