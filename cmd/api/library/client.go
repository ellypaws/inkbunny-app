package library

import (
	"github.com/ellypaws/inkbunny/api"
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var client *http.Client

func (h *Host) NewClient(u *api.Credentials) *http.Client {
	c := http.DefaultClient
	c.Jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if u != nil {
		c.Jar.SetCookies((*url.URL)(h),
			[]*http.Cookie{
				{
					Name:  "sid",
					Value: u.Sid,
				},
				{
					Name:  "username",
					Value: u.Username,
				},
				{
					Name:  "user_id",
					Value: u.UserID.String(),
				},
			})
	}
	return c
}

func (h *Host) GetClient(u *api.Credentials) *http.Client {
	if client != nil {
		return client
	}

	client = h.NewClient(u)
	return client
}

func (r *Request) WithClient(c *http.Client) *Request {
	r.Client = c
	return r
}
