package browser

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/headzoo/ut"
)

func TestDownload(t *testing.T) {
	ut.Run(t)

	out := &bytes.Buffer{}
	u, _ := url.Parse("http://i.imgur.com/HW4bJtY.jpg")
	asset := NewImageAsset(u, "", "", "")
	l, err := DownloadAsset(asset, out)
	ut.AssertNil(err)
	ut.AssertGreaterThan(0, int(l))
	ut.AssertEquals(int(l), out.Len())
}

func TestDownloadAsync(t *testing.T) {
	ut.Run(t)

	ch := make(AsyncDownloadChannel, 1)
	u1, _ := url.Parse("http://i.imgur.com/HW4bJtY.jpg")
	u2, _ := url.Parse("http://i.imgur.com/HkPOzEH.jpg")
	asset1 := NewImageAsset(u1, "", "", "")
	asset2 := NewImageAsset(u2, "", "", "")
	out1 := &bytes.Buffer{}
	out2 := &bytes.Buffer{}

	DownloadAssetAsync(asset1, out1, ch)
	DownloadAssetAsync(asset2, out2, ch)

	queue := 2
	for ; queue > 0; queue-- {
		result := <-ch
		ut.AssertGreaterThan(0, int(result.Size))
		if result.Asset == asset1 {
			ut.AssertEquals(int(result.Size), out1.Len())
		} else if result.Asset == asset2 {
			ut.AssertEquals(int(result.Size), out2.Len())
		} else {
			t.Failed()
		}
	}

	close(ch)
	ut.AssertEquals(0, queue)
}
