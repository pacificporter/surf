package browser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/headzoo/ut"
	"github.com/pacificporter/surf/jar"
)

func newBrowser() *Browser {
	return &Browser{
		headers: make(http.Header, 10),
		history: jar.NewMemoryHistory(),
	}
}

func TestBrowserForm(t *testing.T) {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = fmt.Fprint(w, htmlForm)
		} else {
			_ = r.ParseForm()
			_, _ = fmt.Fprint(w, r.Form.Encode())
		}
	}))
	defer ts.Close()

	bow := &Browser{}
	bow.headers = make(http.Header, 10)
	bow.history = jar.NewMemoryHistory()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	v, ok := f.Field("age")
	ut.AssertTrue(ok)
	ut.AssertEquals("", v)

	v, ok = f.Field("ageage")
	ut.AssertFalse(ok)
	ut.AssertEquals("", v)

	ut.AssertNil(f.Input("age", "55"))

	v, ok = f.Field("age")
	ut.AssertTrue(ok)
	ut.AssertEquals("55", v)

	ut.AssertNil(f.Input("gender", "male"))

	ut.AssertNil(f.Click("submit2"))
	ut.AssertContains("age=55", bow.Body())
	ut.AssertContains("gender=male", bow.Body())
	ut.AssertContains("submit2=submitted2", bow.Body())
}

func TestBrowserForm2(t *testing.T) {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = fmt.Fprint(w, htmlForm2)
		} else {
			_ = r.ParseForm()
			_, _ = fmt.Fprint(w, r.Form.Encode())
		}
	}))
	defer ts.Close()

	bow := &Browser{}
	bow.headers = make(http.Header, 10)
	bow.history = jar.NewMemoryHistory()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	v, ok := f.Field("ageage")
	ut.AssertFalse(ok)
	ut.AssertEquals("", v)

	v, ok = f.Field("city")
	ut.AssertTrue(ok)
	ut.AssertEquals("NY", v)

	v, ok = f.Field("not-selected")
	ut.AssertTrue(ok)
	ut.AssertEquals("Kawasaki", v)

	ut.AssertNil(f.Add("ageage", "55"))

	ut.AssertNil(f.Input("gender", "male"))

	ut.AssertNil(f.Click("submit3"))

	ut.AssertContains("ageage=55", bow.Body())
	ut.AssertContains("gender=male", bow.Body())
	ut.AssertContains("submit3=submitted3", bow.Body())

}

var htmlForm = `<!doctype html>
<html>
	<head>
		<title>Echo Form</title>
	</head>
	<body>
		<form method="post" action="/" name="default">
			<input type="text" name="age" value="" />
			<input type="radio" name="gender" value="male" />
			<input type="radio" name="gender" value="female" />
			<input type="submit" name="submit1" value="submitted1" />
			<input type="submit" name="submit2" value="submitted2" />
		</form>
	</body>
</html>
`

var htmlForm2 = `<!doctype html>
<html>
	<head>
		<title>Echo Form</title>
	</head>
	<body>
		<form method="post" action="/" name="default">
			<input type="text" name="will_be_deleted" value="aaa">
			<input type="text" name="company" value="none">
			<input type="text" name="age" value="55">
			<input type="radio" name="gender" value="male" checked>
			<input type="radio" name="gender" value="female">
			<input type="checkbox" name="music" value="jazz" checked="checked">
			<input type="checkbox" name="music" value="rock">
			<input type="checkbox" name="music" value="fusion" checked>
			<select name="city">
				<option value="NY" selected>
				<option value="Tokyo">
			</select>
			<select name="not-selected">
				<option value="Kawasaki">
				<option value="Tokyo">
				<option value="NY">
			</select>
			<textarea name="hobby">Dance</textarea>
			<input type="submit" name="submit3" value="submitted3">
		</form>
	</body>
</html>
`

func TestSubmitMultipart(t *testing.T) {
	ts := setupTestServer(`<!doctype html>
<html>
	<head>
		<title>multipart form</title>
	</head>
	<body>
		<form method="post" action="/" name="default" enctype="multipart/form-data">
			<input type="text" name="comment" value="" />
			<input type="file" name="image" />
			<input type="submit" name="submit" value="submitted1" />
		</form>
	</body>
</html>
`, t)
	defer ts.Close()

	bow := newBrowser()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	err = f.Input("comment", "my profile picture")
	ut.AssertNil(err)

	imgData, err := base64.StdEncoding.DecodeString(image)
	ut.AssertNil(err)
	err = f.File("image", "profile.png", bytes.NewBuffer(imgData))
	ut.AssertNil(err)
	err = f.Submit()
	ut.AssertNil(err)
	ut.AssertContains("comment=my+profile+picture", bow.Body())
	ut.AssertContains("image=profile.png", bow.Body())
	ut.AssertContains(fmt.Sprintf("profile.png=%s", url.QueryEscape(image)), bow.Body())
}

func setupTestServer(html string, t *testing.T) *httptest.Server {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, err := fmt.Fprint(w, html)
			ut.AssertNil(err)
		} else {
			ct := r.Header.Get("Content-Type")
			if strings.LastIndex(ct, "multipart/form-data") != -1 {

				err := r.ParseMultipartForm(1024 * 1024) // max 1MB ram
				ut.AssertNil(err)
				values := url.Values{}
				for k, av := range r.MultipartForm.Value {
					for _, v := range av {
						values.Add(k, v)
					}
				}
				for k, af := range r.MultipartForm.File {
					for _, fh := range af {
						values.Add(k, fh.Filename)
						f, _ := fh.Open()
						data, _ := ioutil.ReadAll(f)
						val := base64.StdEncoding.EncodeToString(data)
						values.Add(fh.Filename, val)
					}
				}
				_, err = fmt.Fprint(w, values.Encode())
				ut.AssertNil(err)
			} else {
				err := r.ParseForm()
				ut.AssertNil(err)
				_, err = fmt.Fprint(w, r.Form.Encode())
				ut.AssertNil(err)
			}

		}
	}))

	return ts
}

var image = `iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAACjUlEQVRYR+2Wy6oiMRCG4x0V76CIim504fu/gk/hQlS8LhQVERR15stQPZl0a3IOwtlMgXSnU/nrq0p12thoNHqqH7T4D8bWof8DeFXg+fzTJnKldK57c/7dNjsBYrGY4hcFkUwmVSqV0vMmFL7y7F1w5pIuh8fjoeLxv5y5XE61Wi1VKpUUANj9fleHw0Etl0t1Op0CSR8QJ4BkD0i321XtdltXw8wQkGq1qmq1moaYTqeuvIJ5J4B4djodHVy2xI4gQM1mUwPOZjNdOVcvOHuAQJQdAF9ji4rFojM4el4ACPo2lWwZa3zMCUAJy+Wyj5b2wZ/SFwqFf5r3lYATIJ1OB93+SsR8LhUAIpPJOJc4AUTB1Ux2pFfNavs5AW63m+IVlMxsAXssryjXy+ViT4fGTgAC73Y7vdCnEfEBmIPJp2pOAMTW67UW456rCJtj7mUM7Gq1CmUb9cAJQEYcr4vFQldAgnA1zazOdrtV+/3eq2JOAIKQ8Xw+11lJL3A1g8rebzYbNZlMAtiorM1nTgBzH8nseDxGvt9SKbZLIF3BmY/Z/wklEzMwZwFHcb1ejwxuBmJrAKVi1+s1mDIrZ/o7P0b5fF71+339PfAxAjUaDX0SjsdjdT6f3y4LbQEC0mAEHw6HOrhZkbeKvyfxZQ1r0cDQNHtGNEIAOCYSCX38DgYD/Y8Hi1osIvZVfFmLBlpo2m8O60IAsrjX66lsNqsDmz87mD22/dFAC4tKIhKAstFwnzK00PQCYP/kb9enAN5phirAJ5TvfxTtd4HQQjPq8xwCqFQqQfCvdP4rONEAAm3bQgC8v5/MXgKiibZtIQCaxaS2F3x1LMmgKWeCqREC4Nj9ROltUDTRtu0X2hs2IkarWoAAAAAASUVORK5CYII=`
