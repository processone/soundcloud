package soundcloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const scClientID = "YOUR_SOUNDCLOUD_CLIENTID"

// TODO Cache URL for a while to limit traffic on SoundCloud
func GetSongID(url string) (songID string, err error) {
	response, err := httpGetWithTimeout(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	doc, err := html.Parse(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	var currentNode *html.Node
	if currentNode, err = matchTag(doc, "html"); err != nil {
		log.Fatal("Cannot find html tag")
	}
	if currentNode, err = matchTag(currentNode, "head"); err != nil {
		log.Fatal("Cannot find head tag")
	}

	for h := currentNode.FirstChild; h != nil; h = h.NextSibling {
		if h.Data == "meta" {
			var found bool
			var v string
			for _, a := range h.Attr {
				if a.Key == "property" && a.Val == "al:ios:url" {
					found = true
				}
				if a.Key == "content" {
					v = a.Val
				}
			}
			// TODO: Check that the content is actually a SoundCloud soundId
			if found {
				i := strings.LastIndex(v, ":")
				songID = v[i+1:]
				break
			}
		}
	}
	return
}

type streamURLResponse struct {
	status   string
	location string
}

// If the http client follow the redirect, we can just use permanent URL format to play stream
func FormatStreamURL(songID string) string {
	return fmt.Sprintf("https://api.soundcloud.com/tracks/%s/stream?client_id=%s", songID, scClientID)
}

// If the client does not follow redirect, we can use API to extract temporary stream URL
func GetStreamURL(songID string) (stream string, err error) {
	url := fmt.Sprintf("https://api.soundcloud.com/tracks/%s/stream?client_id=%s", songID, scClientID)
	fmt.Println(url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := http.DefaultTransport.RoundTrip(req)
	defer response.Body.Close()

	var r streamURLResponse

	resp, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(resp))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = json.Unmarshal(resp, &r); err != nil {
		fmt.Println("cannot unmarshal json: " + err.Error())
		return
	}
	stream = r.location
	return
}

func printBody(b io.ReadCloser) {
	contents, err := ioutil.ReadAll(b)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(contents))
}

func matchTag(node *html.Node, t string) (h *html.Node, err error) {
	for h = node.FirstChild; h != nil; h = h.NextSibling {
		if h.Type == html.ElementNode && h.Data == t {
			return
		}
	}
	err = errors.New("tag not found: " + t)
	return
}

func httpGetWithTimeout(url string) (*http.Response, error) {
	timeout := time.Duration(15 * time.Second)
	c := http.Client{Timeout: timeout}
	return c.Get(url)
}
