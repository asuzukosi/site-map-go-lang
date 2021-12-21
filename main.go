package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/html"
)

/*
	1. Get the web page
	2. Parse all the links on a page
	3. Build proper urls with our links
	4. Filter out links with a different domain
	5. Find all the pages
	6. Print out xml
*/

// Set the xml name space
const XMLNameSpace = "http://www.sitemaps.org/schemas/sitemap/0.9/"

// A Link is a link in a html document
type Link struct {
	Href string
	Text string
}

type loc struct {
	Value string `xml:"loc"`
}

type urlSet struct {
	URLs  []loc  `xml:"urls"`
	Xmlns string `xml:"xmlns,attr"`
}

func main() {
	urlFlag := flag.String("url", "http://gophercises.com", "the url that you want to build a site map for")
	maxDepth := flag.Int("depth", 3, "max depth of the links traversal")

	flag.Parse()

	hrefs := TraversWebPage(*urlFlag, *maxDepth)

	var mySet urlSet = urlSet{
		Xmlns: XMLNameSpace,
	}

	for _, href := range hrefs {
		mySet.URLs = append(mySet.URLs, loc{href})
	}
	fmt.Print(xml.Header)
	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("", "  ")
	if err := enc.Encode(mySet); err != nil {
		panic(err)
	}
	fmt.Println()

}

// Parse will take in a html document and will returne a slice of
// links or an error
func Parse(r io.Reader) ([]Link, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var links []Link = []Link{}
	nodes := dfs(doc)

	for _, node := range nodes {
		links = append(links, buildLink(node))
	}

	return links, nil
}

func buildLink(n *html.Node) Link {
	var ret Link
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			ret.Href = attr.Val
			break
		}
	}
	ret.Text = text(n)
	return ret
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type != html.ElementNode {
		return ""
	}
	var ret string = ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret += text(c) + " "
	}
	return strings.Join(strings.Fields(ret), " ")
}

func dfs(n *html.Node) []*html.Node {
	if n.Type == html.ElementNode && n.Data == "a" {
		return []*html.Node{n}
	}
	var ret []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret = append(ret, dfs(c)...)
	}
	return ret
}

func filter(keepFunc func(string) bool, links []string) []string {
	var ret []string
	for _, link := range links {
		if keepFunc(link) {
			ret = append(ret, link)
		}
	}
	return ret
}

func withPrefix(pref string) func(string) bool {
	return func(link string) bool {
		return strings.HasPrefix(link, pref)
	}

}
func getLinks(urlPath string) []string {
	resp, err := http.Get(urlPath)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	reqUrl := resp.Request.URL
	baseUrl := &url.URL{
		Scheme: reqUrl.Scheme,
		Host:   reqUrl.Host,
	}
	base := baseUrl.String()

	hrefs := getHrefs(resp.Body, base)
	filterFunc := withPrefix(base)
	return filter(filterFunc, hrefs)
}
func getHrefs(r io.Reader, base string) []string {
	links, _ := Parse(r)
	var hrefs []string
	for _, link := range links {
		// if !strings.HasSuffix(link.Href, "/") {
		// 	link.Href = link.Href + "/"
		// }
		switch {
		case strings.HasPrefix(link.Href, "/"):
			hrefs = append(hrefs, base+link.Href)

		case strings.HasPrefix(link.Href, "http"):
			hrefs = append(hrefs, link.Href)
		}

	}
	return hrefs
}

func TraversWebPage(rootUrl string, depth int) []string {
	var currentList []string = []string{rootUrl}
	var tempList []string = []string{}
	var fullList []string = []string{}
	for i := 0; i <= depth; i++ {
		for _, link := range currentList {
			tempList = append(tempList, getLinks(link)...)
		}

		currentList = tempList
		fullList = updateSet(fullList, currentList)
	}
	return fullList
}

func updateSet(set []string, new_list []string) []string {
	for _, item := range new_list {
		if !isInList(set, item) {
			set = append(set, item)
		}
	}
	return set
}

func isInList(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}
