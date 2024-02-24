package main

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/fulldump/goconfig"
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
		Languages: "en,es,zh",
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

				f.WriteString(`<!DOCTYPE html>` + "\n")
				f.WriteString(`<html lang="` + variation.Language + `">` + "\n")
				f.WriteString(`<head>` + "\n")
				f.WriteString(`<title>` + variation.Title + `</title>` + "\n")
				f.WriteString(`</head>` + "\n")
				f.WriteString(`<body>` + "\n")

				f.WriteString(
					fmt.Sprintln(`<!--`, variation.Url, variation.Language, variation.Filename, variation.Version, `-->`),
				)

				{ // top bar
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
				}

				fmt.Fprintln(f, `<style>
.tree {
  float: left;
}

.tree .children {
  padding-left: 16px;
}

.tree .children {
    display: none;
}

.tree .item.active + .children {
    display: block;
}

.tree .item.selected {
	font-weight: bold;
}

.index {
  float: right;
  position: sticky;
  top: 0;
  padding-top: 16px;
  margin-top: 16px;
}

.index a {
  display: block;
  text-decoration: none;
  color: gray;
  padding: 4px 8px;
}

.index .index-h2 a {
  padding-left: 16px;
}

.index .index-h3 a {
  padding-left: 32px;
}

.index .index-h4 a {
  padding-left: 48px;
}

.index .index-h5 a {
  padding-left: 64px;
}

.index .index-h6 {
  padding-left: 80px;
}

.index a {
  border-left: solid silver 1px;
}

.index a.active {
  border-left: solid black 3px;
  margin-left: -1px;
  color: black;
  font-weight: bold;
}

.index a:hover {
  zborder-left: solid black 3px;
  zmargin-left: -1px;
  color: black;
  text-decoration: underline;
}

.content {
  padding-left: 200px;
  min-height: 100vh;
}

.content .alert {
  color: dodgerblue;
  background-color: ;
  border: solid dodgerblue 1px;
  padding: 16px;
  border-radius: 4px;
}

.breadcrumb .arrow {
  color: silver;
}

.breadcrumb .item {
  color: gray;
  text-decoration: none;
}

.breadcrumb .item:hover {
  color: black;
  text-decoration: underline 1px silver;
}

.breadcrumb .item.selected {
  color: black;
}

.top {
  border-bottom: solid silver 1px;
  margin-bottom: 16px;
}

.footer {
  background-color: #333;
  color: white;
  padding: 32px;
  text-align: center;
  min-height: 500px;
}

.home-desc {
  display: none;
}

</style>`)

				{ // tree
					f.WriteString(`<div class="tree">` + "\n")
					index := getIndex(root, node, language, version)
					f.WriteString(index)
					f.WriteString(`</div>` + "\n")
				}

				{ // content
					f.WriteString(`<div class="content">` + "\n")

					breadcrumb := getBreadcrumb(node, language, version)
					f.WriteString(breadcrumb)

					if variation.Version != "" && version > variation.Version { // todo: make this comparison better (taking into account numbers, not only strings)
						fmt.Fprintln(f, `<div class="alert">This has been unchanged since version `+variation.Version+`</div>`)
					}

					src, err := os.Open(variation.Filename)
					if err != nil {
						panic(err.Error())
					}
					doc := &html.Node{
						Type:     html.ElementNode,
						Data:     "body",
						DataAtom: atom.Body,
					}
					nodes, err := html.ParseFragment(src, doc)
					if err != nil {
						panic(err.Error())
					}

					{ // print index

						onThisPage := ""

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
							})
						}

						if onThisPage != "" {
							f.WriteString(`<div class="index">` + "\n")
							f.WriteString(`On this page:` + "\n")
							f.WriteString(onThisPage)
							f.WriteString(`</div>` + "\n")
						}
					}

					{ // print content
						for _, n := range nodes {
							html.Render(f, n)
						}
					}

					// 					fmt.Fprintln(f, `<!-- begin wwww.htmlcommentbox.com -->
					//  <div id="HCB_comment_box" style="height: auto;"><a href="http://www.htmlcommentbox.com">Widget</a> is loading comments...</div>
					//  <link rel="stylesheet" type="text/css" href="https://www.htmlcommentbox.com/static/skins/bootstrap/twitter-bootstrap.css?v=0" />
					//  <script type="text/javascript" id="hcb"> /*<!--*/ if(!window.hcb_user){hcb_user={};} (function(){var s=document.createElement("script"), l=hcb_user.PAGE || (""+window.location).replace(/'/g,"%27"), h="https://www.htmlcommentbox.com";s.setAttribute("type","text/javascript");s.setAttribute("src", h+"/jread?page="+encodeURIComponent(l).replace("+","%2B")+"&mod=%241%24wq1rdBcg%24zriY6QFS8E0rsWG5aZV1n."+"&opts=16798&num=10&ts=1708504120915");if (typeof s!="undefined") document.getElementsByTagName("head")[0].appendChild(s);})(); /*-->*/ </script>
					// <!-- end www.htmlcommentbox.com -->`)

					f.WriteString(`</div>` + "\n")
				}

				{ // footer
					f.WriteString(`<div class="footer">` + "\n")
					f.WriteString(`HolaDoc` + "\n")
					f.WriteString(`</div>` + "\n")
				}

				fmt.Fprintln(f, `<script>

// source: https://stackoverflow.com/questions/49958471/highlight-item-in-an-index-based-on-currently-visible-content-during-scroll
function isElementInViewport (el) {
    
    // //special bonus for those using jQuery
    // if (typeof $ === "function" && el instanceof $) {
    //     el = el[0];
    // }
		
    var rect     = el.getBoundingClientRect(),
        vWidth   = window.innerWidth || doc.documentElement.clientWidth,
        vHeight  = window.innerHeight || doc.documentElement.clientHeight,
        efp      = function (x, y) { return document.elementFromPoint(x, y) };     

    // Return false if it's not in the viewport
    if (rect.right < 0 || rect.bottom < 0 
            || rect.left > vWidth || rect.top > vHeight)
        return false;

    // Return true if any of its four corners are visible
    return (
          el.contains(efp(rect.left,  rect.top))
      ||  el.contains(efp(rect.right, rect.top))
      ||  el.contains(efp(rect.right, rect.bottom))
      ||  el.contains(efp(rect.left,  rect.bottom))
    );
}

function highlightIndex() {
	let v = false;
	document.querySelectorAll('.index a').forEach(a => {
		const el = document.getElementById(a.getAttribute('href').slice(1));
		
		if (!v && isElementInViewport(el)) {
			a.classList.add('active');	
			v = true;
		} else {
			a.classList.remove('active');	
		}
	});
}

document.addEventListener('scroll', highlightIndex, true);
document.addEventListener('load', highlightIndex, true);
</script>`)

				f.WriteString(`</body>` + "\n")
				f.WriteString(`</html>` + "\n")

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

func getBreadcrumb(n *Node, lang, version string) string {
	breadcrumb := []*Node{}

	for n != nil && len(n.Variations) > 0 {
		breadcrumb = append(breadcrumb, n)
		n = n.Parent
	}

	slices.Reverse(breadcrumb)

	result := ""

	result += `<div class="breadcrumb">`
	for i, node := range breadcrumb {
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
	Title    string
}

func (n *Node) PrettyPrint(indent int) {
	for _, child := range n.Children {
		fmt.Printf("%s (%d)\n", strings.Repeat("    ", indent)+child.Name, len(child.Variations))
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

			filename := path.Join(src, entry.Name())

			f, err := os.Open(filename)

			doc, err := html.Parse(f)
			f.Close()
			if err != nil {
				panic(err.Error())
			}

			title := ""
			traverseHtml(doc, func(node *html.Node) {
				if node.Data == "h1" && node.FirstChild != nil {
					title = node.FirstChild.Data
				}
			})
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

func traverseHtml(n *html.Node, callback func(node *html.Node)) {
	if n.Type == html.ElementNode {
		callback(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {

		traverseHtml(c, callback)
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
