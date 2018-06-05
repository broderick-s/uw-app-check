/*
Used to send an email of the acceptance page for UW

Takes in:
UW username / UW password / email

Could be cleaned up a bit and organzied properly but it's a one off thing.
It's a crude email that is send as the needed message isn't in a specific
element. (not sure where it'll be)
Created by Broderick Stadden
*/
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

type postData struct {
	url  string
	data url.Values
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Need args: UW username / UW password / email ")
	}
	user := os.Args[1]
	pass := os.Args[2]
	e := os.Args[3]

	appURL := "http://sdb.admin.uw.edu/admissions/uwnetid/appstatus.asp"
	post, err := getPostSuffix(appURL)
	if err != nil {
		log.Fatal(err)
	}
	postURL := "https://idp.u.washington.edu" + post

	options := cookiejar.Options{PublicSuffixList: publicsuffix.List}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}

	httpClient := http.Client{Jar: jar}

	resp, err := httpClient.PostForm(postURL, url.Values{
		"j_username":       {user},
		"j_password":       {pass},
		"_eventId_proceed": {"Sign in"},
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err = httpClient.Get(appURL)
	if err != nil {
		log.Fatal(err)
	}

	data := getVeriPostFormData(resp)
	resp, err = httpClient.PostForm(data.url, data.data)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = httpClient.Get(appURL)
	if err != nil {
		log.Fatal(err)
	}

	pageData, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	body, err := getBodyContent(string(pageData))
	if err != nil {
		log.Fatal(err)
	}

	from := mail.NewEmail("Self", e)
	subject := "UW Application Update"
	to := mail.NewEmail("Example User", e)
	message := mail.NewSingleEmail(from, subject, to, appURL, body)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}

func getPostSuffix(url string) (string, error) {
	resp, _ := http.Get(url)
	t := html.NewTokenizer(resp.Body)
	defer resp.Body.Close()

	for {
		ht := t.Next()

		switch {
		case ht == html.ErrorToken:
			return "", errors.New("No form found")
		case ht == html.StartTagToken:
			tok := t.Token()
			if tok.Data == "form" {
				return getAttr("action", tok.Attr), nil
			}
		}
	}
}

func getBodyContent(htmlStr string) (string, error) {
	r := strings.NewReader(htmlStr)
	str := NewSkipTillReader(r, []byte("<BODY>"))
	rtr := NewReadTillReader(str, []byte("</BODY>"))
	bs, err := ioutil.ReadAll(rtr)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func getVeriPostFormData(r *http.Response) postData {
	t := html.NewTokenizer(r.Body)
	defer r.Body.Close()

	data := postData{url: "", data: url.Values{}}
	for {
		ht := t.Next()

		switch {
		case ht == html.ErrorToken:
			return data
		case ht == html.StartTagToken:
			tok := t.Token()
			if tok.Data == "form" {
				data.url = getAttr("action", tok.Attr)
				continue
			}
		case ht == html.SelfClosingTagToken:
			tok := t.Token()
			if tok.Data == "input" {
				k, v := getInputKeyVal(tok.Attr)
				data.data.Add(k, v)
				continue
			}
		}
	}
}

// Returns value of attribute by type
func getAttr(t string, attrs []html.Attribute) string {
	for _, attr := range attrs {
		if attr.Key == t {
			return attr.Val
		}
	}
	return ""
}

func getInputKeyVal(attrs []html.Attribute) (string, string) {
	var key, val string
	for _, attr := range attrs {
		if attr.Key == "name" {
			key = attr.Val
		}
		if attr.Key == "value" {
			val = attr.Val
		}
	}
	return key, val
}
