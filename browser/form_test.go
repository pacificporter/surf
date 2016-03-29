package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/headzoo/ut"
	"github.com/pacificporter/surf/jar"
)

func TestBrowserForm(t *testing.T) {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, htmlForm)
		} else {
			_ = r.ParseForm()
			fmt.Fprint(w, r.Form.Encode())
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
			fmt.Fprint(w, htmlForm2)
		} else {
			_ = r.ParseForm()
			fmt.Fprint(w, r.Form.Encode())
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
