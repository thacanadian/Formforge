package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var titleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
var ogTitleRE = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)=["']og:title["'][^>]+content=["']([^"']+)["']`)

func classifyCreatorURL(u *url.URL) string {
	h := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(h, "youtube.com") || h == "youtu.be":
		return "youtube"
	case strings.Contains(h, "tiktok.com"):
		return "tiktok"
	case strings.Contains(h, "instagram.com"):
		return "instagram"
	case strings.Contains(h, "spotify.com"):
		return "spotify"
	case strings.Contains(h, "podcasts.apple.com"):
		return "podcast"
	case strings.Contains(h, "threads.net"):
		return "threads"
	case h == "x.com" || strings.Contains(h, "twitter.com"):
		return "x"
	case strings.Contains(h, "facebook.com"):
		return "facebook"
	default:
		return "website"
	}
}

func validatePublicURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("enter a valid public HTTP or HTTPS URL")
	}
	h := strings.ToLower(u.Hostname())
	if h == "localhost" || strings.HasSuffix(h, ".local") {
		return nil, errors.New("local and private-network links are not allowed")
	}
	ips, err := net.LookupIP(h)
	if err == nil {
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				return nil, errors.New("local and private-network links are not allowed")
			}
		}
	}
	u.User = nil
	u.Fragment = ""
	return u, nil
}

func previewPublicCreatorURL(raw string) (CoachLink, error) {
	u, err := validatePublicURL(raw)
	if err != nil {
		return CoachLink{}, err
	}
	link := CoachLink{Platform: classifyCreatorURL(u), URL: u.String(), Provider: u.Hostname(), AddedAt: nowISO()}
	client := &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 4 {
			return errors.New("too many redirects")
		}
		_, e := validatePublicURL(req.URL.String())
		return e
	}}
	var endpoint string
	switch link.Platform {
	case "youtube":
		endpoint = "https://www.youtube.com/oembed?format=json&url=" + url.QueryEscape(u.String())
	case "tiktok":
		endpoint = "https://www.tiktok.com/oembed?url=" + url.QueryEscape(u.String())
	case "spotify":
		endpoint = "https://open.spotify.com/oembed?url=" + url.QueryEscape(u.String())
	}
	if endpoint != "" {
		if resp, e := client.Get(endpoint); e == nil {
			defer resp.Body.Close()
			if resp.StatusCode < 300 {
				var d struct {
					Title        string `json:"title"`
					AuthorName   string `json:"author_name"`
					ThumbnailURL string `json:"thumbnail_url"`
				}
				_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d)
				link.Title = strings.TrimSpace(d.Title)
				link.Handle = strings.TrimSpace(d.AuthorName)
				link.ThumbnailURL = strings.TrimSpace(d.ThumbnailURL)
			}
		}
	}
	if link.Title == "" && link.Platform == "website" {
		req, _ := http.NewRequest("GET", u.String(), nil)
		req.Header.Set("User-Agent", "FormForge/1.4 link preview")
		if resp, e := client.Do(req); e == nil {
			defer resp.Body.Close()
			if resp.StatusCode < 300 {
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				if m := ogTitleRE.FindSubmatch(b); len(m) > 1 {
					link.Title = strings.TrimSpace(string(m[1]))
				} else if m := titleRE.FindSubmatch(b); len(m) > 1 {
					link.Title = strings.TrimSpace(string(m[1]))
				}
			}
		}
	}
	if link.Title == "" {
		link.Title = u.Hostname()
	}
	return link, nil
}

func initialsFor(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "?"
	}
	out := ""
	for _, p := range parts {
		out += strings.ToUpper(p[:1])
		if len(out) == 2 {
			break
		}
	}
	return out
}

func validateCustomCoach(in CustomCoachProfile) (CustomCoachProfile, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	in.Summary = strings.TrimSpace(in.Summary)
	if len(in.Name) < 2 || len(in.Name) > 100 {
		return in, errors.New("creator name must be 2–100 characters")
	}
	if in.Category == "" {
		in.Category = "Custom creator"
	}
	if len(in.Summary) < 20 || len(in.Summary) > 2000 {
		return in, errors.New("add an original 20–2,000 character summary")
	}
	if len(in.Principles) == 0 {
		return in, errors.New("add at least one coaching principle")
	}
	if len(in.Principles) > 12 || len(in.Communication) > 12 || len(in.Links) > 12 {
		return in, errors.New("custom profiles support up to 12 principles, traits, and links")
	}
	in.Initials = initialsFor(in.Name)
	if in.Licensed {
		in.Status = "official"
	} else {
		in.Status = "editorial"
	}
	if in.SafetyNote == "" {
		in.SafetyNote = "User-added editorial profile. It does not imply endorsement, identity imitation, or permission to copy protected content."
	}
	for i, l := range in.Links {
		u, err := validatePublicURL(l.URL)
		if err != nil {
			return in, fmt.Errorf("link %d: %w", i+1, err)
		}
		l.URL = u.String()
		l.Platform = classifyCreatorURL(u)
		l.AddedAt = nowISO()
		in.Links[i] = l
	}
	return in, nil
}
