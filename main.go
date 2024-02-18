package main

import (
	"cmp"
	"fmt"
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
		for _, variation := range node.Variations {

			newFilename := path.Join(c.Www, getOutputPath(node, variation)+".html")

			os.MkdirAll(path.Dir(newFilename), 0777) // todo: handle err

			f, err := os.Create(newFilename)
			if err != nil {
				panic(err.Error())
			}

			_, err = f.WriteString(
				fmt.Sprintln(variation.Url, variation.Language, variation.Filename, variation.Version),
			)
			if err != nil {
				panic(err.Error())
			}

			err = f.Close()
			if err != nil {
				panic(err.Error())
			}
		}
	})

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

	result = append([]string{variation.Language, variation.Version}, result...)

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
