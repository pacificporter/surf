package browser

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/itchio/go-brotli/dec"
	"github.com/pacificporter/surf/errors"
	"github.com/pacificporter/surf/jar"
)

// Attribute represents a Browser capability.
type Attribute int

// AttributeMap represents a map of Attribute values.
type AttributeMap map[Attribute]bool

type File struct {
	fileName string
	data     io.Reader
}

type FileSet map[string]*File

const (
	// SendRefererAttribute instructs a Browser to send the Referer header.
	SendReferer Attribute = iota

	// MetaRefreshHandlingAttribute instructs a Browser to handle the refresh meta tag.
	MetaRefreshHandling

	// FollowRedirectsAttribute instructs a Browser to follow Location headers.
	FollowRedirects
)

// InitialAssetsArraySize is the initial size when allocating a slice of page
// assets. Increasing this size may lead to a very small performance increase
// when downloading assets from a page with a lot of assets.
var InitialAssetsSliceSize = 20

// Browsable represents an HTTP web browser.
type Browsable interface {
	// SetUserAgent sets the user agent.
	SetUserAgent(ua string)

	// SetAttribute sets a browser instruction attribute.
	SetAttribute(a Attribute, v bool)

	// SetAttributes is used to set all the browser attributes.
	SetAttributes(a AttributeMap)

	// SetBookmarksJar sets the bookmarks jar the browser uses.
	SetBookmarksJar(bj jar.BookmarksJar)

	// SetCookieJar is used to set the cookie jar the browser uses.
	SetCookieJar(cj http.CookieJar)

	// SetHistoryJar is used to set the history jar the browser uses.
	SetHistoryJar(hj jar.History)

	// SetHeadersJar sets the headers the browser sends with each request.
	SetHeadersJar(h http.Header)

	// SetTimeout sets the timeout for requests.
	SetTimeout(t time.Duration)

	// SetTransport sets the Transport of the browser. It can be used for use Proxy.
	SetTransport(t *http.Transport)

	// AddRequestHeader adds a header the browser sends with each request.
	AddRequestHeader(name, value string)

	// SetRequestHeader sets a header the browser sends with each request.
	SetRequestHeader(name, value string)

	// Open requests the given URL using the GET method.
	Open(url string) error

	// OpenForm appends the data values to the given URL and sends a GET request.
	OpenForm(url string, data url.Values) error

	// OpenBookmark calls Get() with the URL for the bookmark with the given name.
	OpenBookmark(name string) error

	// Post requests the given URL using the POST method.
	Post(url string, contentType string, body io.Reader) error

	// PostForm requests the given URL using the POST method with the given data.
	PostForm(url string, data url.Values) error

	// PostMultipart requests the given URL using the POST method with the given data using multipart/form-data format.
	PostMultipart(u string, data url.Values, files FileSet) error

	// Back loads the previously requested page.
	Back() bool

	// Reload duplicates the last successful request.
	Reload() error

	// Bookmark saves the page URL in the bookmarks with the given name.
	Bookmark(name string) error

	// Click clicks on the page element matched by the given expression.
	Click(expr string) error

	// Form returns the form in the current page that matches the given expr.
	Form(expr string) (Submittable, error)

	// Forms returns an array of every form in the page.
	Forms() []Submittable

	// Links returns an array of every link found in the page.
	Links() []*Link

	// Images returns an array of every image found in the page.
	Images() []*Image

	// Stylesheets returns an array of every stylesheet linked to the document.
	Stylesheets() []*Stylesheet

	// Scripts returns an array of every script linked to the document.
	Scripts() []*Script

	// SiteCookies returns the cookies for the current site.
	SiteCookies() []*http.Cookie

	// ResolveUrl returns an absolute URL for a possibly relative URL.
	ResolveUrl(u *url.URL) *url.URL

	// ResolveStringUrl works just like ResolveUrl, but the argument and return value are strings.
	ResolveStringUrl(u string) (string, error)

	// Download writes the contents of the document to the given writer.
	Download(o io.Writer) (int64, error)

	// Url returns the page URL as a string.
	Url() *url.URL

	// StatusCode returns the response status code.
	StatusCode() int

	// Title returns the page title.
	Title() string

	// ResponseHeaders returns the page headers.
	ResponseHeaders() http.Header

	// Response returns a http.Response pointer.
	Response() *http.Response

	// Body returns the page body as a string of html.
	Body() string

	// Dom returns the inner *goquery.Selection.
	Dom() *goquery.Selection

	// Find returns the dom selections matching the given expression.
	Find(expr string) *goquery.Selection
}

// Default is the default Browser implementation.
type Browser struct {
	// state is the current browser state.
	state *jar.State

	// userAgent is the User-Agent header value sent with requests.
	userAgent string

	// cookies stores cookies for every site visited by the browser.
	cookies http.CookieJar

	// bookmarks stores the saved bookmarks.
	bookmarks jar.BookmarksJar

	// history stores the visited pages.
	history jar.History

	// headers are additional headers to send with each request.
	headers http.Header

	// attributes is the set browser attributes.
	attributes AttributeMap

	// refresh is a timer used to meta refresh pages.
	refresh *time.Timer

	// timeout is the browser timeout
	timeout time.Duration

	// transport is the browser connection transport.
	transport *http.Transport

	// body of the current page.
	body []byte
}

// Open requests the given URL using the GET method.
func (bow *Browser) Open(u string) error {
	ur, err := url.Parse(u)
	if err != nil {
		return err
	}
	return bow.httpGET(ur, nil)
}

// OpenForm appends the data values to the given URL and sends a GET request.
func (bow *Browser) OpenForm(u string, data url.Values) error {
	ul, err := url.Parse(u)
	if err != nil {
		return err
	}
	ul.RawQuery = data.Encode()

	return bow.Open(ul.String())
}

// OpenBookmark calls Open() with the URL for the bookmark with the given name.
func (bow *Browser) OpenBookmark(name string) error {
	url, err := bow.bookmarks.Read(name)
	if err != nil {
		return err
	}
	return bow.Open(url)
}

// Post requests the given URL using the POST method.
func (bow *Browser) Post(u string, contentType string, body io.Reader) error {
	ur, err := url.Parse(u)
	if err != nil {
		return err
	}
	return bow.httpPOST(ur, nil, contentType, body)
}

// PostForm requests the given URL using the POST method with the given data.
func (bow *Browser) PostForm(u string, data url.Values) error {
	return bow.Post(u, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// PostMultipart requests the given URL using the POST method with the given data using multipart/form-data format.
func (bow *Browser) PostMultipart(u string, data url.Values, files FileSet) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, vs := range data {
		for _, v := range vs {
			writer.WriteField(k, v)
		}
	}
	for k, file := range files {
		fw, err := writer.CreateFormFile(k, file.fileName)
		if err != nil {
			return err
		}
		if file.data != nil {
			_, err := io.Copy(fw, file.data)
			if err != nil {
				return err
			}
		}
	}
	err := writer.Close()
	if err != nil {
		return err

	}
	return bow.Post(u, writer.FormDataContentType(), body)
}

// Back loads the previously requested page.
//
// Returns a boolean value indicating whether a previous page existed, and was
// successfully loaded.
func (bow *Browser) Back() bool {
	if bow.history.Len() > 1 {
		bow.state = bow.history.Pop()
		return true
	}
	return false
}

// Reload duplicates the last successful request.
func (bow *Browser) Reload() error {
	if bow.state.Request != nil {
		return bow.httpRequest(bow.state.Request)
	}
	return errors.NewPageNotLoaded("Cannot reload, the previous request failed.")
}

// Bookmark saves the page URL in the bookmarks with the given name.
func (bow *Browser) Bookmark(name string) error {
	return bow.bookmarks.Save(name, bow.ResolveUrl(bow.Url()).String())
}

// Click clicks on the page element matched by the given expression.
//
// Currently this is only useful for click on links, which will cause the browser
// to load the page pointed at by the link. Future versions of Surf may support
// JavaScript and clicking on elements will fire the click event.
func (bow *Browser) Click(expr string) error {
	sel := bow.Find(expr)
	if sel.Length() == 0 {
		return errors.NewElementNotFound(
			"Element not found matching expr '%s'.", expr)
	}
	if !sel.Is("a") {
		return errors.NewElementNotFound(
			"Expr '%s' must match an anchor tag.", expr)
	}

	href, err := bow.attrToResolvedUrl("href", sel)
	if err != nil {
		return err
	}

	return bow.httpGET(href, bow.Url())
}

// Form returns the form in the current page that matches the given expr.
func (bow *Browser) Form(expr string) (Submittable, error) {
	sel := bow.Find(expr)
	if sel.Length() == 0 {
		return nil, errors.NewElementNotFound(
			"Form not found matching expr '%s'.", expr)
	}
	if !sel.Is("form") {
		return nil, errors.NewElementNotFound(
			"Expr '%s' does not match a form tag.", expr)
	}

	return NewForm(bow, sel), nil
}

// Forms returns an array of every form in the page.
func (bow *Browser) Forms() []Submittable {
	sel := bow.Find("form")
	len := sel.Length()
	if len == 0 {
		return nil
	}

	forms := make([]Submittable, len)
	sel.Each(func(i int, s *goquery.Selection) {
		forms[i] = NewForm(bow, s)
	})
	return forms
}

// Links returns an array of every link found in the page.
func (bow *Browser) Links() []*Link {
	links := make([]*Link, 0, InitialAssetsSliceSize)
	bow.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, err := bow.attrToResolvedUrl("href", s)
		if err == nil {
			links = append(links, NewLinkAsset(
				href,
				bow.attrOrDefault("id", "", s),
				s.Text(),
			))
		}
	})

	return links
}

// Images returns an array of every image found in the page.
func (bow *Browser) Images() []*Image {
	images := make([]*Image, 0, InitialAssetsSliceSize)
	bow.Find("img").Each(func(_ int, s *goquery.Selection) {
		src, err := bow.attrToResolvedUrl("src", s)
		if err == nil {
			images = append(images, NewImageAsset(
				src,
				bow.attrOrDefault("id", "", s),
				bow.attrOrDefault("alt", "", s),
				bow.attrOrDefault("title", "", s),
			))
		}
	})

	return images
}

// Stylesheets returns an array of every stylesheet linked to the document.
func (bow *Browser) Stylesheets() []*Stylesheet {
	stylesheets := make([]*Stylesheet, 0, InitialAssetsSliceSize)
	bow.Find("link").Each(func(_ int, s *goquery.Selection) {
		rel, ok := s.Attr("rel")
		if ok && rel == "stylesheet" {
			href, err := bow.attrToResolvedUrl("href", s)
			if err == nil {
				stylesheets = append(stylesheets, NewStylesheetAsset(
					href,
					bow.attrOrDefault("id", "", s),
					bow.attrOrDefault("media", "all", s),
					bow.attrOrDefault("type", "text/css", s),
				))
			}
		}
	})

	return stylesheets
}

// Scripts returns an array of every script linked to the document.
func (bow *Browser) Scripts() []*Script {
	scripts := make([]*Script, 0, InitialAssetsSliceSize)
	bow.Find("script").Each(func(_ int, s *goquery.Selection) {
		src, err := bow.attrToResolvedUrl("src", s)
		if err == nil {
			scripts = append(scripts, NewScriptAsset(
				src,
				bow.attrOrDefault("id", "", s),
				bow.attrOrDefault("type", "text/javascript", s),
			))
		}
	})

	return scripts
}

// SiteCookies returns the cookies for the current site.
func (bow *Browser) SiteCookies() []*http.Cookie {
	return bow.cookies.Cookies(bow.Url())
}

// SetCookieJar is used to set the cookie jar the browser uses.
func (bow *Browser) SetCookieJar(cj http.CookieJar) {
	bow.cookies = cj
}

// SetUserAgent sets the user agent.
func (bow *Browser) SetUserAgent(userAgent string) {
	bow.userAgent = userAgent
}

// SetAttribute sets a browser instruction attribute.
func (bow *Browser) SetAttribute(a Attribute, v bool) {
	bow.attributes[a] = v
}

// SetAttributes is used to set all the browser attributes.
func (bow *Browser) SetAttributes(a AttributeMap) {
	bow.attributes = a
}

// SetBookmarksJar sets the bookmarks jar the browser uses.
func (bow *Browser) SetBookmarksJar(bj jar.BookmarksJar) {
	bow.bookmarks = bj
}

// SetHistoryJar is used to set the history jar the browser uses.
func (bow *Browser) SetHistoryJar(hj jar.History) {
	bow.history = hj
}

// SetHeadersJar sets the headers the browser sends with each request.
func (bow *Browser) SetHeadersJar(h http.Header) {
	bow.headers = h
}

// SetTransport sets the http library transport mechanism for each request.
// SetTimeout sets the timeout for requests.
func (bow *Browser) SetTimeout(t time.Duration) {
	bow.timeout = t
}

// SetTransport sets the Transport of the browser. It can be used for set Proxy.
func (bow *Browser) SetTransport(t *http.Transport) {
	bow.transport = t
}

// AddRequestHeader sets a header the browser sends with each request.
func (bow *Browser) AddRequestHeader(name, value string) {
	bow.headers.Add(name, value)
}

// SetRequestHeader sets a header the browser sends with each request.
func (bow *Browser) SetRequestHeader(name, value string) {
	bow.headers.Set(name, value)
}

// ResolveUrl returns an absolute URL for a possibly relative URL.
func (bow *Browser) ResolveUrl(u *url.URL) *url.URL {
	return bow.Url().ResolveReference(u)
}

// ResolveStringUrl works just like ResolveUrl, but the argument and return value are strings.
func (bow *Browser) ResolveStringUrl(u string) (string, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	pu = bow.Url().ResolveReference(pu)
	return pu.String(), nil
}

// Download writes the contents of the document to the given writer.
func (bow *Browser) Download(o io.Writer) (int64, error) {
	h, err := bow.state.Dom.Html()
	if err != nil {
		return 0, err
	}
	l, err := io.WriteString(o, h)
	return int64(l), err
}

// Url returns the page URL as a string.
func (bow *Browser) Url() *url.URL {
	if bow.state == nil || bow.state.Response == nil {
		return &url.URL{}
	}
	return bow.state.Request.URL
}

// StatusCode returns the response status code.
func (bow *Browser) StatusCode() int {
	if bow.state == nil || bow.state.Response == nil {
		return 0
	}
	return bow.state.Response.StatusCode
}

// Title returns the page title.
func (bow *Browser) Title() string {
	if bow.state == nil || bow.state.Dom == nil {
		return ""
	}
	return bow.state.Dom.Find("title").Text()
}

// ResponseHeaders returns the page headers.
func (bow *Browser) ResponseHeaders() http.Header {
	if bow.state == nil || bow.state.Response == nil {
		return http.Header{}
	}
	return bow.state.Response.Header
}

// Response returns the pointer to http.Response.
func (bow *Browser) Response() *http.Response {
	if bow.state == nil {
		return &http.Response{}
	}
	return bow.state.Response
}

// Body returns the page body as a string of html.
func (bow *Browser) Body() string {
	if bow.state == nil || bow.state.Dom == nil {
		return ""
	}
	body, _ := bow.state.Dom.Find("body").Html()
	return body
}

// Dom returns the inner *goquery.Selection.
func (bow *Browser) Dom() *goquery.Selection {
	if bow.state == nil || bow.state.Dom == nil {
		return &goquery.Selection{}
	}
	return bow.state.Dom.First()
}

// Find returns the dom selections matching the given expression.
func (bow *Browser) Find(expr string) *goquery.Selection {
	if bow.state == nil || bow.state.Dom == nil {
		return &goquery.Selection{}
	}
	return bow.state.Dom.Find(expr)
}

// -- Unexported methods --

// buildClient creates, configures, and returns a *http.Client type.
func (bow *Browser) buildClient() *http.Client {
	client := &http.Client{}
	client.Jar = bow.cookies
	client.CheckRedirect = bow.shouldRedirect

	if bow.transport != nil {
		client.Transport = bow.transport
	}
	client.Timeout = bow.timeout

	return client
}

// buildRequest creates and returns a *http.Request type.
// Sets any headers that need to be sent with the request.
func (bow *Browser) buildRequest(method, url string, ref *url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = bow.headers
	req.Header.Set("User-Agent", bow.userAgent)
	if bow.attributes[SendReferer] && ref != nil {
		req.Header.Set("Referer", ref.String())
	}

	return req, nil
}

// httpGET makes an HTTP GET request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) httpGET(u *url.URL, ref *url.URL) error {
	req, err := bow.buildRequest("GET", u.String(), ref, nil)
	if err != nil {
		return err
	}
	return bow.httpRequest(req)
}

// httpPOST makes an HTTP POST request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) httpPOST(u *url.URL, ref *url.URL, contentType string, body io.Reader) error {
	req, err := bow.buildRequest("POST", u.String(), ref, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	return bow.httpRequest(req)
}

// send uses the given *http.Request to make an HTTP request.
func (bow *Browser) httpRequest(req *http.Request) (err error) {
	bow.preSend()
	resp, err := bow.buildClient().Do(req)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("Response is nil")
	}

	var reader io.ReadCloser
	var isWrapped bool
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		isWrapped = true
	case "deflate":
		reader = flate.NewReader(resp.Body)
		isWrapped = true
	case "br":
		reader = dec.NewBrotliReader(resp.Body)
		isWrapped = true
	default:
		reader = resp.Body
	}
	defer func() {
		if isWrapped {
			if cerr := reader.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	bow.body, err = ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	buff := bytes.NewBuffer(bow.body)
	dom, err := goquery.NewDocumentFromReader(buff)
	if err != nil {
		return err
	}

	bow.history.Push(bow.state)
	bow.state = jar.NewHistoryState(req, resp, dom)
	bow.postSend()

	return nil
}

// preSend sets browser state before sending a request.
func (bow *Browser) preSend() {
	if bow.refresh != nil {
		bow.refresh.Stop()
	}
}

// postSend sets browser state after sending a request.
func (bow *Browser) postSend() {
	if bow.attributes[MetaRefreshHandling] {
		sel := bow.Find("meta[http-equiv='refresh']")
		if sel.Length() > 0 {
			attr, ok := sel.Attr("content")
			if ok {
				dur, err := time.ParseDuration(attr + "s")
				if err == nil {
					bow.refresh = time.NewTimer(dur)
					go func() {
						<-bow.refresh.C
						_ = bow.Reload()
					}()
				}
			}
		}
	}
}

// shouldRedirect is used as the value to http.Client.CheckRedirect.
func (bow *Browser) shouldRedirect(req *http.Request, _ []*http.Request) error {
	if bow.attributes[FollowRedirects] {
		return nil
	}
	return errors.NewLocation(
		"Redirects are disabled. Cannot follow '%s'.", req.URL.String())
}

// attributeToUrl reads an attribute from an element and returns a url.
func (bow *Browser) attrToResolvedUrl(name string, sel *goquery.Selection) (*url.URL, error) {
	src, ok := sel.Attr(name)
	if !ok {
		return nil, errors.NewAttributeNotFound(
			"Attribute '%s' not found.", name)
	}
	ur, err := url.Parse(src)
	if err != nil {
		return nil, err
	}

	return bow.ResolveUrl(ur), nil
}

// attributeOrDefault reads an attribute and returns it or the default value when it's empty.
func (bow *Browser) attrOrDefault(name, def string, sel *goquery.Selection) string {
	a, ok := sel.Attr(name)
	if ok {
		return a
	}
	return def
}
