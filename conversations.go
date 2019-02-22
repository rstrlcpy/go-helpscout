package helpscout

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ConversationLister interface {
	Process(c Conversation) bool
}

type ConditionType int

const (
	Inclusively ConditionType = 1
	Exclusively ConditionType = 2
)

const (
	ByUser     = "user"
	ByCustomer = "customer"
)

const (
	ConversationTypeEmail = "email"
	ConversationTypeChat  = "chat"
	ConversationTypePhone = "phone"
)

const (
	ConversationStatusOpen    = "open"
	ConversationStatusClosed  = "closed"
	ConversationStatusActive  = "active"
	ConversationStatusPending = "pending"
	ConversationStatusSpam    = "spam"
)

const (
	ConversationStatePublished = "published"
	ConversationStateDraft     = "draft"
	ConversationStateDeleted   = "deleted"
)

type filterUintValues struct {
	cType  ConditionType
	values []uint
}

func (v *filterUintValues) Set(values []uint, cType ConditionType) {
	v.values = values
	v.cType = cType
}

type filterStringValues struct {
	cType  ConditionType
	values []string
}

func (v *filterStringValues) Set(values []string, cType ConditionType) {
	v.values = values
	v.cType = cType
}

type filterTimePeriod struct {
	cType ConditionType
	from  time.Time
	to    time.Time
}

func (v *filterTimePeriod) Set(from time.Time, to time.Time, cType ConditionType) {
	v.from = from
	v.to = to
	v.cType = cType
}

type ConversationLookupFilter struct {
	mailboxIds    *filterUintValues
	statuses      *filterStringValues
	types         *filterStringValues
	states        *filterStringValues
	createdPeriod *filterTimePeriod
	updatedPeriod *filterTimePeriod
}

func NewConversationLookupFilter() *ConversationLookupFilter {
	return &ConversationLookupFilter{}
}

func getConditionType(cType []ConditionType) ConditionType {
	if len(cType) == 0 {
		return Inclusively
	}

	if len(cType) > 1 {
		panic("There must be only one condition type")
	}

	return cType[0]
}

func (f *ConversationLookupFilter) MailboxIds(ids []uint, cType ...ConditionType) {
	if f.mailboxIds == nil {
		f.mailboxIds = &filterUintValues{}
	}
	f.mailboxIds.Set(ids, getConditionType(cType))
}

func (f *ConversationLookupFilter) Status(statuses []string, cType ...ConditionType) {
	if f.statuses == nil {
		f.statuses = &filterStringValues{}
	}
	f.statuses.Set(statuses, getConditionType(cType))
}

func (f *ConversationLookupFilter) State(states []string, cType ...ConditionType) {
	if f.states == nil {
		f.states = &filterStringValues{}
	}
	f.states.Set(states, getConditionType(cType))
}

func (f *ConversationLookupFilter) Type(types []string, cType ...ConditionType) {
	if f.types == nil {
		f.types = &filterStringValues{}
	}
	f.types.Set(types, getConditionType(cType))
}

func (f *ConversationLookupFilter) CreatedTime(from time.Time, to time.Time, cType ...ConditionType) {
	if f.createdPeriod == nil {
		f.createdPeriod = &filterTimePeriod{}
	}
	f.createdPeriod.Set(from, to, getConditionType(cType))
}

func (f *ConversationLookupFilter) ModifiedTime(from time.Time, to time.Time, cType ...ConditionType) {
	if f.updatedPeriod == nil {
		f.updatedPeriod = &filterTimePeriod{}
	}
	f.updatedPeriod.Set(from, to, getConditionType(cType))
}

type Conversation struct {
	Id        uint      `json:"id"`
	Number    uint      `json:"number"`
	Threads   uint      `json:"threads"`
	Type      string    `json:"type"`
	FolderId  uint      `json:"folderId"`
	Status    string    `json:"status"`
	State     string    `json:"state"`
	Subject   string    `json:"subject"`
	Preview   string    `json:"preview"`
	MailboxId uint      `json:"mailboxId"`
	Assignee  User      `json:"assignee"`
	CreatedBy User      `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	ClosedAt  time.Time `json:"closedAt"`
	UpdatedAt time.Time `json:"userUpdatedAt"`
	ClosedBy  uint      `json:"closedBy"`
	Answered  struct {
		Time               time.Time `json:"time"`
		FriendlyWaitPeriod string    `json:"friendly"`
		By                 string    `json:"latestReplyFrom"`
	} `json:"customerWaitingSince"`
	Source struct {
		Via  string `json:"via"`
		Type string `json:"type"`
	} `json:"source"`
	Tags            []TagShort `json:"tags"`
	CC              []string   `json:"cc"`
	BCC             []string   `json:"bcc"`
	PrimaryCustomer struct {
		Id uint `json:"id"`
	} `json:"primaryCustomer"`
	CustomFields []CustomField `json:"customFields"`
}

type ConversationThread struct {
	Type     string   `json:"type"`
	Body     string   `json:"text"`
	Cc       []string `json:"cc"`
	Bcc      []string `json:"bcc"`
	Customer struct {
		Email string `json:"email"`
	} `json:"customer"`
}

type ConversationCreateRequest struct {
	Type     string `json:"type"`
	Customer struct {
		Email string `json:"email"`
	} `json:"customer"`
	Subject   string               `json:"subject"`
	MailboxId uint                 `json:"mailboxId"`
	Tags      []string             `json:"tags"`
	Status    string               `json:"status"`
	CreatedBy uint                 `json:"user"`
	Threads   []ConversationThread `json:"threads"`
}

func (c *Client) CreateConversation(sender User, mailboxId uint,
	to []string, cc []string, bcc []string, subject string, body string) error {

	conversation := ConversationCreateRequest{
		Type:      ConversationTypeEmail,
		Status:    ConversationStatusActive,
		CreatedBy: sender.Id,
		Tags:      []string{"upstream"},
		Threads:   make([]ConversationThread, 1),
		Subject:   subject,
		MailboxId: mailboxId,
	}

	conversation.Customer.Email = to[0]

	conversation.Threads[0] = ConversationThread{
		Type: ThreadTypeReply,
		Body: body,
		Cc:   append(cc, to[1:]...),
		Bcc:  bcc,
	}

	conversation.Threads[0].Customer.Email = to[0]

	return c.doApiCall(http.MethodPost, "/conversations", nil, &conversation, nil)
}

func (c *Client) ListConversations(filter *ConversationLookupFilter, lister ConversationLister) error {
	query, err := prepareListConversationQuery(filter)
	if err != nil {
		return err
	}

	if filter.statuses == nil {
		return c.listConversationsImpl(query, lister)
	}

	statuses := prepareListOfStatuses(filter)
	for _, status := range statuses {
		query.Set("status", status)
		err := c.listConversationsImpl(query, lister)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) listConversationsImpl(query *url.Values, lister ConversationLister) error {
	page := 1
	query.Del("page")
	for {
		var cList struct {
			Conversations []Conversation `json:"conversations"`
		}

		req := &generalListApiCallReq{
			Embedded: &cList,
		}
		err := c.doApiCall(http.MethodGet, "/conversations", query, nil, req)
		if err != nil {
			return err
		}

		if req.Page.TotalPages == 0 {
			break
		}

		for _, conversation := range cList.Conversations {
			if !lister.Process(conversation) {
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

func prepareListOfStatuses(filter *ConversationLookupFilter) []string {
	var statuses []string
	if filter.statuses != nil {
		switch filter.statuses.cType {
		case Inclusively:
			statuses = append(statuses, filter.statuses.values...)
		case Exclusively:
			m := map[string]int{
				ConversationStatusOpen:    0,
				ConversationStatusClosed:  0,
				ConversationStatusActive:  0,
				ConversationStatusPending: 0,
				ConversationStatusSpam:    0,
			}

			for _, v := range filter.statuses.values {
				delete(m, v)
			}

			for k, _ := range m {
				statuses = append(statuses, k)
			}
		default:
			panic("Unknown condition type")
		}
	}

	return statuses
}

func prepareListConversationQuery(filter *ConversationLookupFilter) (*url.Values, error) {
	if filter == nil {
		return &url.Values{}, nil
	}

	queryValues := []string{}

	if filter.mailboxIds != nil {
		b := make([]string, len(filter.mailboxIds.values))
		for i, v := range filter.mailboxIds.values {
			b[i] = fmt.Sprintf("mailboxid:%d", v)
		}

		queryValues = append(queryValues, fmt.Sprintf("(%s)", strings.Join(b, " OR ")))
	}

	if filter.createdPeriod != nil {
		fromStr, toStr := formatFromToTimePeriod(filter.createdPeriod.from, filter.createdPeriod.to)
		queryValues = append(queryValues, fmt.Sprintf("createdAt:[%s TO %s]", fromStr, toStr))
	}

	if filter.updatedPeriod != nil {
		fromStr, toStr := formatFromToTimePeriod(filter.updatedPeriod.from, filter.updatedPeriod.to)
		queryValues = append(queryValues, fmt.Sprintf("modifiedAt:[%s TO %s]", fromStr, toStr))
	}

	query := url.Values{}
	if len(queryValues) != 0 {
		query.Set("query", fmt.Sprintf("(%s)", strings.Join(queryValues, " AND ")))
	}

	return &query, nil
}

func formatFromToTimePeriod(from time.Time, to time.Time) (string, string) {
	fromStr := "*"
	if !from.IsZero() {
		fromStr = from.Format("2006-01-02T15:04:05Z")
	}

	toStr := "*"
	if !to.IsZero() {
		toStr = to.Format("2006-01-02T15:04:05Z")
	}

	return fromStr, toStr
}
