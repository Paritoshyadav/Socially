package service

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

var queriesCache = make(map[string]*template.Template)

const (
	minPageSize     = 1
	defaultPageSize = 10
	maxPageSize     = 99
)

func isUnquieViolation(err error) bool {

	pgerr, ok := err.(*pgconn.PgError)

	return ok && pgerr.Code == pgerrcode.UniqueViolation
}

func isforeignKeyViolation(err error) bool {

	pgerr, ok := err.(*pgconn.PgError)

	return ok && pgerr.Code == pgerrcode.ForeignKeyViolation
}

func buildQuery(query string, data map[string]interface{}) (string, []interface{}, error) {

	t, ok := queriesCache[query]

	if !ok {
		var err error
		t, err = template.New("query").Parse(query)
		if err != nil {
			return "", nil, fmt.Errorf("could not able to parse sql query in template: %v", err)
		}
		queriesCache[query] = t
	}

	var wr bytes.Buffer

	if err := t.Execute(&wr, data); err != nil {
		return "", nil, fmt.Errorf("could not apply query to template: %v", err)
	}
	args := []interface{}{}

	q := wr.String()

	for key, val := range data {
		if !strings.Contains(q, "@"+key) {
			continue
		}
		args = append(args, val)
		q = strings.Replace(q, "@"+key, fmt.Sprintf("$%d", len(args)), -1)
	}

	return q, args, nil
}

func normalizePageSize(i int) int {
	if i == 0 {
		return defaultPageSize
	}
	if i < minPageSize {
		return minPageSize
	}
	if i > maxPageSize {
		return maxPageSize
	}
	return i
}
