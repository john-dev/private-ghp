package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"private-ghp/config"
	"strings"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func setupHttpHandler() {
	config := config.GetConfig()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if page, ok := findPage(r, config); !ok {
			logrus.Debugf("page not found for request %s", r.Host)
			http.Error(w, "Page Not Found", http.StatusNotFound)
			return
		} else {
			logrus.Debugf("page found for request %s%s", r.Host, r.RequestURI)
			cookie, err := r.Cookie("token")
			if err != nil {
				redirectURL := fmt.Sprintf(
					"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=http://%s/login/github/callback?origin=http://%s.%s/%s&scope=repo", config.Github.Client.Id, config.Url, page.Subdomain, config.Url, r.RequestURI)
				logrus.Debugf("no cookie found for request, redirecting to %s", redirectURL)
				http.Redirect(w, r, redirectURL, 301)
			} else {
				logrus.Debugf("cookie found for request %s", r.Host)
				ts := oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: cookie.Value},
				)

				tc := oauth2.NewClient(ctx, ts)
				client = github.NewClient(tc)

				if r.RequestURI == "/" {
					r.RequestURI = "/" + page.Index
				}
				c, _, _, err := client.Repositories.GetContents(ctx, page.Repository.Owner, page.Repository.Name, r.RequestURI, &github.RepositoryContentGetOptions{Ref: page.Repository.Branch})
				logrus.Debugf("requesting content for owner: %s, repo: %s, path: %s, branch: %s", page.Repository.Owner, page.Repository.Name, r.RequestURI, page.Repository.Branch)
				if err == nil {
					logrus.Debugf("content found for owner: %s, repo: %s, path: %s, branch: %s", page.Repository.Owner, page.Repository.Name, r.RequestURI, page.Repository.Branch)
					sDec, _ := base64.StdEncoding.DecodeString(*c.Content)
					setContentType(c, w, r.RequestURI)
					w.Write(sDec)
				} else {
					logrus.Debugf("content not found for owner: %s, repo: %s, path: %s, branch: %s", page.Repository.Owner, page.Repository.Name, r.RequestURI, page.Repository.Branch)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}

		}
	})

	http.HandleFunc("/login/github/callback", func(w http.ResponseWriter, r *http.Request) {
		origin := r.URL.Query().Get("origin")
		code := r.URL.Query().Get("code")
		token := getGithubAccessToken(code, config)

		w.Header().Add("Set-Cookie", fmt.Sprintf("token=%s; Domain=.%s; Path=/; HttpOnly", token, config.Url))
		redirectURL := origin
		logrus.Debugf("token recevied from github, redirecting to %s", redirectURL)
		http.Redirect(w, r, redirectURL, 301)
	})
}

func findPage(r *http.Request, config *config.Config) (*config.Page, bool) {
	for _, page := range config.Pages {
		if fmt.Sprintf("%s.%s", page.Subdomain, config.Url) == r.Host {
			return &page, true
		}
	}
	return nil, false
}

func setContentType(c *github.RepositoryContent, w http.ResponseWriter, uri string) {
	t := "text/plain"

	if strings.HasSuffix(uri, ".html") || strings.HasSuffix(uri, ".htm") {
		t = "text/html"
	}
	if strings.HasSuffix(uri, ".css") {
		t = "text/css"
	}
	if strings.HasSuffix(uri, ".js") {
		t = "application/javascript"
	}
	if strings.HasSuffix(uri, ".jpg") || strings.HasSuffix(uri, ".jpeg") {
		t = "image/jpeg"
	}
	if strings.HasSuffix(uri, ".gif") {
		t = "image/gif"
	}
	if strings.HasSuffix(uri, ".png") {
		t = "image/png"
	}

	w.Header().Add("Content-Type", t)
}
