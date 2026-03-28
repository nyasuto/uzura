package page

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/html"
)

func (p *Page) emitFailed(reqID, url string, ts float64, errText string) {
	p.emit(NetworkEvent{
		Type:      NetworkLoadingFailed,
		RequestID: reqID,
		URL:       url,
		Timestamp: ts,
		ErrorText: errText,
	})
}

func (p *Page) handleFulfill(reqID, url string, now func() float64, result *InterceptResult) error {
	statusCode := result.ResponseCode
	if statusCode == 0 {
		statusCode = 200
	}
	respHeaders := make(http.Header)
	for k, v := range result.ResponseHeaders {
		respHeaders.Set(k, v)
	}

	p.emit(NetworkEvent{
		Type:        NetworkResponseReceived,
		RequestID:   reqID,
		URL:         url,
		Timestamp:   now(),
		StatusCode:  statusCode,
		StatusText:  http.StatusText(statusCode),
		MimeType:    respHeaders.Get("Content-Type"),
		RespHeaders: respHeaders,
	})
	p.emit(NetworkEvent{
		Type:              NetworkLoadingFinished,
		RequestID:         reqID,
		Timestamp:         now(),
		EncodedDataLength: int64(len(result.ResponseBody)),
		Body:              result.ResponseBody,
	})

	doc, err := html.Parse(bytes.NewReader(result.ResponseBody))
	if err != nil {
		return fmt.Errorf("navigate parse fulfilled %s: %w", url, err)
	}
	doc.SetQueryEngine(css.NewEngine())
	p.doc = doc
	p.url = url
	return nil
}

func (p *Page) fetchWithOverrides(ctx context.Context, url string, headers http.Header) (*http.Response, error) {
	if headers == nil {
		return p.fetcher.FetchContext(ctx, url)
	}
	return p.fetcher.FetchContextWithHeaders(ctx, url, headers)
}

func mimeFromResponse(resp *http.Response) string {
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		return "text/html"
	}
	for i, c := range ct {
		if c == ';' {
			return ct[:i]
		}
		_ = i
	}
	return ct
}
