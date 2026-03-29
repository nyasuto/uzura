package page

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/network"
)

// Navigate loads the document at the given URL.
// It performs: robots check → fetch → decode → parse → DOM construction.
func (p *Page) Navigate(ctx context.Context, url string) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPageClosed
	}
	pageCtx := p.ctx
	interceptor := p.requestInterceptor
	p.mu.Unlock()

	// Merge caller context with page context: canceled if either is done.
	mergedCtx, mergedCancel := mergeContexts(ctx, pageCtx)
	defer mergedCancel()
	ctx = mergedCtx

	reqID := fmt.Sprintf("req-%d", requestCounter.Add(1))
	now := nowFunc()

	p.emit(NetworkEvent{
		Type:      NetworkRequestWillBeSent,
		RequestID: reqID,
		URL:       url,
		Method:    http.MethodGet,
		Timestamp: now(),
	})

	fetchURL := url
	var headerOverrides http.Header

	// Request interception hook.
	if interceptor != nil {
		result, iErr := interceptor(ctx, InterceptedRequest{
			RequestID: reqID,
			URL:       url,
			Method:    http.MethodGet,
			Stage:     StageRequest,
		})
		if iErr != nil {
			p.emitFailed(reqID, url, now(), iErr.Error())
			return fmt.Errorf("navigate %s: intercept: %w", url, iErr)
		}
		if result != nil {
			switch result.Action {
			case InterceptFail:
				reason := result.ErrorReason
				if reason == "" {
					reason = "BlockedByClient"
				}
				p.emitFailed(reqID, url, now(), reason)
				return fmt.Errorf("navigate %s: blocked by client (%s)", url, reason)
			case InterceptFulfill:
				return p.handleFulfill(reqID, url, now, result)
			case InterceptContinue:
				if result.URL != "" {
					fetchURL = result.URL
				}
				if result.Headers != nil {
					headerOverrides = result.Headers
				}
			}
		}
	}

	resp, err := p.fetchWithOverrides(ctx, fetchURL, headerOverrides)
	if err != nil {
		p.emitFailed(reqID, url, now(), err.Error())
		return fmt.Errorf("navigate %s: %w", url, err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	respHeaders := resp.Header
	respURL := resp.Request.URL.String()
	mimeType := mimeFromResponse(resp)

	reader, err := network.DecodeResponse(resp)
	if err != nil {
		return fmt.Errorf("navigate decode %s: %w", url, err)
	}
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("navigate read %s: %w", url, err)
	}

	// Response-stage interception.
	if interceptor != nil {
		result, iErr := interceptor(ctx, InterceptedRequest{
			RequestID:       reqID,
			URL:             fetchURL,
			Method:          http.MethodGet,
			Stage:           StageResponse,
			StatusCode:      statusCode,
			ResponseHeaders: respHeaders,
			Body:            bodyBytes,
		})
		if iErr != nil {
			p.emitFailed(reqID, url, now(), iErr.Error())
			return fmt.Errorf("navigate %s: response intercept: %w", url, iErr)
		}
		if result != nil {
			switch result.Action {
			case InterceptFail:
				reason := result.ErrorReason
				if reason == "" {
					reason = "BlockedByClient"
				}
				p.emitFailed(reqID, url, now(), reason)
				return fmt.Errorf("navigate %s: blocked by client (%s)", url, reason)
			case InterceptFulfill:
				return p.handleFulfill(reqID, url, now, result)
			case InterceptContinue:
				if result.RespStatusCode > 0 {
					statusCode = result.RespStatusCode
				}
				if result.RespHeaders != nil {
					for k, v := range result.RespHeaders {
						respHeaders.Set(k, v)
					}
				}
			}
		}
	}

	p.emit(NetworkEvent{
		Type:        NetworkResponseReceived,
		RequestID:   reqID,
		URL:         respURL,
		Timestamp:   now(),
		StatusCode:  statusCode,
		StatusText:  http.StatusText(statusCode),
		MimeType:    mimeType,
		RespHeaders: respHeaders,
	})

	p.emit(NetworkEvent{
		Type:              NetworkLoadingFinished,
		RequestID:         reqID,
		Timestamp:         now(),
		EncodedDataLength: int64(len(bodyBytes)),
		Body:              bodyBytes,
	})

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("navigate parse %s: %w", url, err)
	}

	doc.SetQueryEngine(css.NewEngine())

	p.mu.Lock()
	p.doc = doc
	p.url = url
	p.mu.Unlock()
	return nil
}

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

	p.mu.Lock()
	p.doc = doc
	p.url = url
	p.mu.Unlock()
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

func nowFunc() func() float64 {
	return func() float64 {
		return float64(time.Now().UnixMilli()) / 1000.0
	}
}

// mergeContexts returns a context that is canceled when either parent is done.
func mergeContexts(ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	merged, cancel := context.WithCancel(ctx1)
	go func() {
		select {
		case <-ctx2.Done():
			cancel()
		case <-merged.Done():
		}
	}()
	return merged, cancel
}
