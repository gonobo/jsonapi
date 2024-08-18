package page

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gonobo/jsonapi/v1/query"
)

var (
	DefaultPageParser   = PageNavigationParser{}
	DefaultCursorParser = CursorNavigationParser{}
)

type Criteria = query.Page

type Params map[string]string

func (p Params) Cursor() string {
	return p[query.ParamPageCursor]
}

func (p Params) Limit() (int, error) {
	value := p[query.ParamPageLimit]
	if value == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(value)
	return limit, err
}

func (p Params) PageNumber() (int, error) {
	value := p[query.ParamPageNumber]
	if value == "" {
		return 0, nil
	}
	pageNumber, err := strconv.Atoi(value)
	return pageNumber, err
}

type PageNavigationParser struct{}

func (p PageNavigationParser) ParsePageQuery(r *http.Request) (Criteria, error) {
	criteria := Criteria{}
	params := make(Params)
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	if limit, err := params.Limit(); err != nil {
		return criteria, fmt.Errorf("parse page limit: %s", err)
	} else {
		criteria.Limit = limit
	}

	if pageNumber, err := params.PageNumber(); err != nil {
		return criteria, fmt.Errorf("parse page number: %s", err)
	} else {
		criteria.PageNumber = pageNumber
	}

	return criteria, nil
}

type CursorNavigationParser struct{}

func (c CursorNavigationParser) ParsePageQuery(r *http.Request) (Criteria, error) {
	criteria := Criteria{}
	params := make(Params)
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	if limit, err := params.Limit(); err != nil {
		return criteria, fmt.Errorf("parse page limit: %s", err)
	} else {
		criteria.Limit = limit
	}

	criteria.Cursor = params.Cursor()
	return criteria, nil
}
