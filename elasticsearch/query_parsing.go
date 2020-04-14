package elasticsearch

import (
	"crypto/sha1"
	"fmt"
	"github.com/apex/log"
	"github.com/tidwall/gjson"
	"strings"
)

func ParseJsonBodyLines(body string) []string {
	return strings.Split(strings.Trim(strings.TrimSpace(body), "\n"), "\n")
}

func DeDuplicateJsonLines(queryLines []gjson.Result) []gjson.Result {
	if len(queryLines) <= 1 {
		return queryLines
	}

	var result []gjson.Result

	seen := make(map[string]gjson.Result)

	for _, val := range queryLines {
		valHash := GetHash(val.String())

		if _, ok := seen[valHash]; !ok {
			result = append(result, val)
			seen[valHash] = val
		} else {
			log.Debug("De-duplicated JSON line")
		}
	}

	return result
}

func GetHash(data string) string {
	h := sha1.New()
	h.Write([]byte(data))

	return fmt.Sprintf("%x", h.Sum(nil))
}

func ParseQueries(body string) []gjson.Result {
	var results []gjson.Result

	// Elasticsearch queries can have multiple queries in a single request
	queries := ParseJsonBodyLines(body)

	for _, queryString := range queries {

		if !gjson.Valid(queryString) {
			continue
		}

		queryLine := gjson.Parse(queryString)

		if !IsValidQuery(queryLine) {
			log.WithField("query", queryLine.String()).Debug("Skipping query line (does not satisfy criteria)")
			continue
		}

		actualQuery := ParseActualQuery(queryLine)

		results = append(results, actualQuery)
	}

	return results
}

func IsValidQuery(fullQuery gjson.Result) bool {
	if fullQuery.Value() == nil {
		return false
	}

	query := fullQuery.Get("query")

	return query.Exists() &&
		query.Type != gjson.True &&
		query.Type != gjson.False &&
		query.Value() != "" &&
		query.Value() != nil
}

func ParseActualQuery(query gjson.Result) gjson.Result {
	// Extract the query, if the actual query terms are nested inside of a function_score then drill down into that

	var actualQuery gjson.Result
	if intoFnScore := query.Get("query.function_score"); intoFnScore.Exists() {
		actualQuery = intoFnScore.Get("query")
	} else {
		actualQuery = query.Get("query")
	}

	return actualQuery
}
