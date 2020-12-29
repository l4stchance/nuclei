package http

import (
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/projectdiscovery/nuclei/v2/pkg/operators/extractors"
	"github.com/projectdiscovery/nuclei/v2/pkg/operators/matchers"
	"github.com/projectdiscovery/nuclei/v2/pkg/output"
	"github.com/projectdiscovery/nuclei/v2/pkg/types"
)

// Match matches a generic data response again a given matcher
func (r *Request) Match(data map[string]interface{}, matcher *matchers.Matcher) bool {
	partString := matcher.Part
	switch partString {
	case "header":
		partString = "all_headers"
	case "all":
		partString = "raw"
	}

	item, ok := data[partString]
	if !ok {
		return false
	}
	itemStr := types.ToString(item)

	switch matcher.GetType() {
	case matchers.StatusMatcher:
		statusCode, ok := data["status_code"]
		if !ok {
			return false
		}
		return matcher.Result(matcher.MatchStatusCode(statusCode.(int)))
	case matchers.SizeMatcher:
		return matcher.Result(matcher.MatchSize(len(itemStr)))
	case matchers.WordsMatcher:
		return matcher.Result(matcher.MatchWords(itemStr))
	case matchers.RegexMatcher:
		return matcher.Result(matcher.MatchRegex(itemStr))
	case matchers.BinaryMatcher:
		return matcher.Result(matcher.MatchBinary(itemStr))
	case matchers.DSLMatcher:
		return matcher.Result(matcher.MatchDSL(data))
	}
	return false
}

// Extract performs extracting operation for a extractor on model and returns true or false.
func (r *Request) Extract(data map[string]interface{}, extractor *extractors.Extractor) map[string]struct{} {
	partString := extractor.Part
	switch partString {
	case "header":
		partString = "all_headers"
	case "all":
		partString = "raw"
	}

	item, ok := data[partString]
	if !ok {
		return nil
	}
	itemStr := types.ToString(item)

	switch extractor.GetType() {
	case extractors.RegexExtractor:
		return extractor.ExtractRegex(itemStr)
	case extractors.KValExtractor:
		return extractor.ExtractKval(data)
	}
	return nil
}

// responseToDSLMap converts a HTTP response to a map for use in DSL matching
func (r *Request) responseToDSLMap(resp *http.Response, host, matched, rawReq, rawResp, body, headers string, duration time.Duration, extra map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{}, len(extra)+8+len(resp.Header)+len(resp.Cookies()))
	for k, v := range extra {
		data[k] = v
	}

	data["host"] = host
	data["matched"] = matched
	if r.options.Options.JSONRequests {
		data["request"] = rawReq
		data["response"] = rawResp
	}

	data["content_length"] = resp.ContentLength
	data["status_code"] = resp.StatusCode

	data["body"] = body
	for _, cookie := range resp.Cookies() {
		data[cookie.Name] = cookie.Value
	}
	for k, v := range resp.Header {
		k = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(k, "-", "_")))
		data[k] = strings.Join(v, " ")
	}
	data["all_headers"] = headers

	if r, err := httputil.DumpResponse(resp, true); err == nil {
		rawString := string(r)
		data["raw"] = rawString
	}
	data["duration"] = duration.Seconds()
	return data
}

// makeResultEvent creates a result event from internal wrapped event
func (r *Request) makeResultEvent(wrapped *output.InternalWrappedEvent) []*output.ResultEvent {
	results := make([]*output.ResultEvent, 0, len(wrapped.OperatorsResult.Matches)+1)

	data := output.ResultEvent{
		TemplateID:       r.options.TemplateID,
		Info:             r.options.TemplateInfo,
		Type:             "http",
		Host:             wrapped.InternalEvent["host"].(string),
		Matched:          wrapped.InternalEvent["matched"].(string),
		Metadata:         wrapped.OperatorsResult.PayloadValues,
		ExtractedResults: wrapped.OperatorsResult.OutputExtracts,
	}
	if r.options.Options.JSONRequests {
		data.Request = wrapped.InternalEvent["request"].(string)
		data.Response = wrapped.InternalEvent["raw"].(string)
	}

	// If we have multiple matchers with names, write each of them separately.
	if len(wrapped.OperatorsResult.Matches) > 0 {
		for k := range wrapped.OperatorsResult.Matches {
			data.MatcherName = k
			results = append(results, &data)
		}
	} else {
		results = append(results, &data)
	}
	return results
}
