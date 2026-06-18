package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

const textFetchRetryCount = 10

func newHTTPClient(opt Options) (*http.Client, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if opt.CustomProxy != "" {
		u, err := url.Parse(opt.CustomProxy)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(u)
	} else if !opt.UseSystemProxy {
		tr.Proxy = nil
	}
	return &http.Client{
		Timeout:   time.Duration(opt.HTTPRequestTimeout * float64(time.Second)),
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func doRequestWithRedirects(ctx context.Context, client *http.Client, method, rawURL string, headers map[string]string, configure func(*http.Request)) (*http.Response, error) {
	currentURL := rawURL
	for redirects := 0; redirects <= 10; redirects++ {
		req, err := http.NewRequestWithContext(ctx, method, currentURL, nil)
		if err != nil {
			return nil, err
		}
		applyHeaders(req, headers)
		if configure != nil {
			configure(req)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			return resp, nil
		}
		location := resp.Header.Get("Location")
		if location == "" {
			return resp, nil
		}
		resp.Body.Close()
		base := req.URL
		next, err := base.Parse(location)
		if err != nil {
			return nil, err
		}
		currentURL = next.String()
		// long: 上游手动跟随跳转，核心是让 Cookie/Authorization/Range 等用户头在 CDN 跳转后仍保持一致。
	}
	return nil, fmt.Errorf("重定向次数过多: %s", rawURL)
}

func fetchText(ctx context.Context, client *http.Client, input string, headers map[string]string) (string, string, error) {
	if strings.HasPrefix(input, "file:") {
		u, err := url.Parse(input)
		if err != nil {
			return "", input, err
		}
		path := fileURLPath(u)
		b, err := os.ReadFile(path)
		if err != nil {
			return "", input, err
		}
		finalURL, err := localFileURL(path)
		if err != nil {
			return string(b), input, nil
		}
		return string(b), finalURL, nil
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		var lastErr error
		for try := 0; try < textFetchRetryCount; try++ {
			text, finalURL, err := fetchHTTPTextOnce(ctx, client, input, headers)
			if err == nil {
				return text, finalURL, nil
			}
			lastErr = err
			if ctx.Err() != nil {
				break
			}
		}
		return "", input, fmt.Errorf("文本请求重试 %d 次后失败: %w", textFetchRetryCount, lastErr)
	}
	b, err := os.ReadFile(input)
	if err != nil {
		return "", input, err
	}
	finalURL, err := localFileURL(input)
	if err != nil {
		return "", input, err
	}
	return string(b), finalURL, nil
}

func fetchHTTPTextOnce(ctx context.Context, client *http.Client, input string, headers map[string]string) (string, string, error) {
	resp, err := doRequestWithRedirects(ctx, client, http.MethodGet, input, headers, nil)
	if err != nil {
		return "", input, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", input, fmt.Errorf("HTTP %d: %s", resp.StatusCode, input)
	}
	body := resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", input, err
		}
		defer gz.Close()
		body = gz
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return "", input, err
	}
	return decodeHTTPText(b, resp.Header.Get("Content-Type")), resp.Request.URL.String(), nil
}

func decodeHTTPText(data []byte, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return string(data)
	}
	label := strings.TrimSpace(params["charset"])
	if label == "" {
		return string(data)
	}
	reader, err := charset.NewReaderLabel(label, bytes.NewReader(data))
	if err != nil {
		return string(data)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return string(data)
	}
	// long: 上游会按响应 charset 解码播放列表；非 UTF-8 清单中的中文轨道名或保存名必须在解析前还原成 Unicode。
	return string(decoded)
}

func fetchBytes(ctx context.Context, client *http.Client, input string, headers map[string]string) ([]byte, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		resp, err := doRequestWithRedirects(ctx, client, http.MethodGet, input, headers, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, input)
		}
		return io.ReadAll(resp.Body)
	}
	if strings.HasPrefix(input, "file:") {
		u, err := url.Parse(input)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(fileURLPath(u))
	}
	return os.ReadFile(input)
}

func applyHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func localFileURL(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// long: 上游把普通本地路径提升为绝对 file URI；这样 m3u8 内的相对分片和 key URI 才会以清单所在目录为基准解析。
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String(), nil
}

func fileURLPath(u *url.URL) string {
	if u.Host != "" && u.Host != "localhost" {
		return "//" + u.Host + u.Path
	}
	return u.Path
}
