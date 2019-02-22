package helpscout

import (
	"net/http"
	"net/url"
	"strconv"
)

type UsersLister interface {
	Process(c User) bool
}

type User struct {
	Id        uint   `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first"`
	LastName  string `json:"last"`
	Email     string `json:"email"`
}

func (c *Client) ListUsers(lister UsersLister) error {
	page := 1
	query := &url.Values{}
	for {
		var uList struct {
			Users []User `json:"users"`
		}

		req := &generalListApiCallReq{
			Embedded: &uList,
		}
		err := c.doApiCall(http.MethodGet, "/users", query, nil, req)
		if err != nil {
			return err
		}

		if req.Page.TotalPages == 0 {
			break
		}

		for _, user := range uList.Users {
			if !lister.Process(user) {
				return ErrorInterrupted
			}
		}

		if req.Page.Number == req.Page.TotalPages {
			break
		}

		page++
		query.Set("page", strconv.Itoa(page))
	}

	return nil
}
