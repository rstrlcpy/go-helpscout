package helpscout

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ThreadLister interface {
	Process(thread Thread) bool
}

const (
	ThreadTypeBeaconchat    = "beaconchat"
	ThreadTypeChat          = "chat"
	ThreadTypeCustomer      = "customer"
	ThreadTypeForwardChild  = "forwardchild"
	ThreadTypeForwardParent = "forwardparent"
	ThreadTypeLineitem      = "lineitem"
	ThreadTypeMessage       = "message"
	ThreadTypeNote          = "note"
	ThreadTypePhone         = "phone"
	ThreadTypeReply         = "reply"
)

const (
	ThreadStatusActive   = "active"
	ThreadStatusClosed   = "closed"
	ThreadStatusNochange = "nochange"
	ThreadStatusPending  = "pending"
	ThreadStatusSpam     = "spam"
)

const (
	ThreadStateDraft     = "draft"
	ThreadStateHidden    = "hidden"
	ThreadStatePublished = "published"
	ThreadStateReview    = "review"
)

type ThreadCreator struct {
	Id        uint   `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first"`
	LastName  string `json:"last"`
	Email     string `json:"email"`
}

type Thread struct {
	Id         uint   `json:"id"`
	Type       string `json:"type"`
	AssignedTo User   `json:"assignedTo"`
	Status     string `json:"status"`
	State      string `json:"state"`
	Body       string `json:"body"`
	Source     struct {
		Via  string `json:"via"`
		Type string `json:"type"`
	} `json:"source"`
	Customer     Customer      `json:"customer"`
	CreatedBy    ThreadCreator `json:"createdBy"`
	SavedReplyId uint          `json:"savedReplyId"`
	To           []string      `json:"to"`
	CC           []string      `json:"cc"`
	BCC          []string      `json:"bcc"`
	CreatedAt    time.Time     `json:"createdAt"`
	OpenedAt     time.Time     `json:"openedAt"`
}

func (c *Client) ListThreads(conversationId uint, lister ThreadLister) error {
	resource := fmt.Sprintf("/conversations/%d/threads", conversationId)

	query := &url.Values{}
	page := 1
	for {
		var tList struct {
			Threads []Thread `json:"threads"`
		}

		req := &generalListApiCallReq{
			Embedded: &tList,
		}

		err := c.doApiCall(http.MethodGet, resource, query, nil, req)
		if err != nil {
			return err
		}

		if req.Page.TotalPages == 0 {
			break
		}

		for _, thread := range tList.Threads {
			if !lister.Process(thread) {
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
