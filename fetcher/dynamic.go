package fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
	"github.com/agux/roprox/util"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/pkg/errors"
)

func jsGetText(sel string) (js string) {
	const funcJS = `function getText(sel) {
				var text = [];
				var elements = document.body.querySelectorAll(sel);

				for(var i = 0; i < elements.length; i++) {
					var current = elements[i];
					//if(current.children.length === 0 && current.textContent.replace(/ |\n/g,'') !== '') {
					// Check the element has no children && that it is not empty
					//	text.push(current.textContent + ',');
					//}
					text.push(current.innerText)
				}
				return text
			 };`

	invokeFuncJS := `var a = getText('` + sel + `'); a;`
	js = strings.Join([]string{funcJS, invokeFuncJS}, " ")
	log.Tracef("javascript: %s", js)
	return js
}

func jsPageBottom() (js string) {
	js = `
		function isPageBottom() {
			var b;
			try {
				b = (window.innerHeight + window.scrollY) >= document.body.offsetHeight;
			} catch(ex1) {
				try {
					b = (window.innerHeight + window.pageYOffset) >= document.body.offsetHeight - 2;
				} catch(ex2) {
					try {
						b = (window.innerHeight + window.scrollY) >= document.body.scrollHeight;
					} catch(ex3) {
						var scrollTop = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
						var scrollHeight = (document.documentElement && document.documentElement.scrollHeight) || document.body.scrollHeight;
						b = (scrollTop + window.innerHeight) >= scrollHeight;
					}
				}
			}
			return b;
		};
		var a = isPageBottom(); a;
	`
	log.Tracef("javascript: %s", js)
	return
}

func jsSelect(sel, val string) (js string) {
	funcJS := `
		function setSelectedIndex(sel, v) {
			var s = document.body.querySelector(sel);
			var o;
			for ( var i = 0; i < s.options.length; i++ ) {
				if ( s.options[i].innerText == v ) {
					o = s.options[i];	
				} else {
					s.options[i].removeAttribute("selected");
				}
			}
			if (o != null) {
				o.setAttribute("selected", "");
				s.value = v;
				s.onchange();
				return 1;
			}
			return 0;
		};
	`
	invokeFuncJS := fmt.Sprintf(`var a = setSelectedIndex('%s','%s'); a;`, sel, val)
	js = strings.Join([]string{funcJS, invokeFuncJS}, " ")
	log.Tracef("javascript: %s", js)
	return js
}

func scrollToBottom(ctx context.Context) (e error) {
	var bottom bool
	for i := 1; true; i++ {
		if e = chromedp.Run(ctx,
			chromedp.KeyEvent(kb.End),
		); e != nil {
			return errors.Wrapf(e, "failed to send kb.End key #%d", i)
		}

		log.Debugf("End key sent #%d", i)

		if e = chromedp.Run(ctx,
			chromedp.Evaluate(jsPageBottom(), &bottom),
		); e != nil {
			return errors.Wrapf(e, "failed to check page bottom #%d", i)
		}

		if bottom {
			//found footer
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
	return
}

func dumpHTML(ctx context.Context, filename string) (e error) {
	html := new(string)
	if e = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			*html, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	); e != nil {
		return errors.Wrap(e, "failed to get document outer HTML")
	}
	htmlPath := filepath.Join(
		conf.Args.WebDriver.WorkingFolder,
		fmt.Sprintf("%s_%s.html", filename, time.Now().Format("20060102_150405")),
	)
	if e = ioutil.WriteFile(
		htmlPath,
		[]byte(*html),
		0644); e != nil {
		return errors.Wrapf(e, "unable to save HTML to %s", htmlPath)
	}
	return
}

func forceViewportEmulation(ctx context.Context) (contentSize *dom.Rect, e error) {
	e = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			// get layout metrics
			_, _, contentSize, err = page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to get layout metrics")
			}

			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to force viewport emulation")
			}

			return nil
		}),
	)
	return
}

func captureScreen(ctx context.Context, filename string, quality int) (e error) {
	var contentSize *dom.Rect
	if contentSize, e = forceViewportEmulation(ctx); e != nil {
		return
	}

	buf := make([]byte, 0, 1024)
	bufptr := &buf
	// capture entire browser viewport, returning png with quality=90
	if e = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			// capture screenshot
			*bufptr, err = page.CaptureScreenshot().
				WithQuality(int64(quality)).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  contentSize.Width,
					Height: contentSize.Height,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to capture screenshot for the page")
			}
			return nil
		}),
	); e != nil {
		return errors.Wrapf(e, "failed to capture screen")
	}
	imgPath := filepath.Join(
		conf.Args.WebDriver.WorkingFolder,
		fmt.Sprintf("%s_%s.png", filename, time.Now().Format("20060102_150405")),
	)
	if e = ioutil.WriteFile(
		imgPath,
		buf,
		0644); e != nil {
		return errors.Wrapf(e, "unable to save image to %s", imgPath)
	}

	return
}

// waitPageLoaded blocks until a target receives a Page.loadEventFired.
func waitPageLoaded(ctx context.Context) error {
	// TODO: this function is inherently racy, as we don't run ListenTarget
	// until after the navigate action is fired. For example, adding
	// time.Sleep(time.Second) at the top of this body makes most tests hang
	// forever, as they miss the load event.
	//
	// However, setting up the listener before firing the navigate action is
	// also racy, as we might get a load event from a previous navigate.
	//
	// For now, the second race seems much more common in real scenarios, so
	// keep the first approach. Is there a better way to deal with this?
	ch := make(chan struct{})
	lctx, cancel := context.WithCancel(ctx)
	chromedp.ListenTarget(lctx, func(ev interface{}) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			cancel()
			close(ch)
		}
	})
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func allocatorOptions(fspec types.FetcherSpec) (o []chromedp.ExecAllocatorOption, rpx *types.ProxyServer){
	proxyMode := fspec.ProxyMode()
	df := fspec.(types.DynamicHTMLFetcher)
	switch proxyMode {
	case types.MasterProxy:
		p := fmt.Sprintf("socks5://%s", conf.Args.Network.MasterProxyAddr)
		log.Debugf("using proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	case types.RotateProxy:
		var e error
		if rpx, e = util.PickProxy(); e != nil {
			log.Fatalf("%s unable to pick rotate proxy: %+v", fspec.UID(), e)
			return
		}
		p := fmt.Sprintf("%s://%s:%s", rpx.Type, rpx.Host, rpx.Port)
		log.Debugf("using proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	case types.RotateGlobalProxy:
		var e error
		if rpx, e = util.PickGlobalProxy(); e != nil {
			log.Fatalf("%s unable to pick global rotate proxy: %+v", fspec.UID(), e)
			return
		}
		p := fmt.Sprintf("%s://%s:%s", rpx.Type, rpx.Host, rpx.Port)
		log.Debugf("using global proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	}
	if types.Direct != proxyMode {
		if ua, e := util.PickUserAgent(); e != nil {
			log.Fatalf("failed to pick user agents from the pool: %+v", e)
		} else {
			o = append(o, chromedp.UserAgent(ua))
		}
	}
	if conf.Args.WebDriver.NoImage {
		o = append(o, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}
	// o = append(o, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)

	for _, opt := range chromedp.DefaultExecAllocatorOptions {
		if reflect.ValueOf(chromedp.Headless).Pointer() == reflect.ValueOf(opt).Pointer() {
			if df.Headless() {
				log.Debug("headless mode is enabled")
			} else {
				log.Debug("ignored headless mode")
				continue
			}
		}
		o = append(o, opt)
	}
	return
}