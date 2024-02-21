package fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

var log = logging.Logger

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

// func forceViewportEmulation(ctx context.Context) (contentSize *dom.Rect, e error) {
// 	e = chromedp.Run(ctx,
// 		chromedp.ActionFunc(func(ctx context.Context) (err error) {
// 			// get layout metrics
// 			_, _, contentSize, err = page.GetLayoutMetrics().Do(ctx)
// 			if err != nil {
// 				return errors.Wrapf(err, "failed to get layout metrics")
// 			}

// 			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

// 			// force viewport emulation
// 			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
// 				WithScreenOrientation(&emulation.ScreenOrientation{
// 					Type:  emulation.OrientationTypePortraitPrimary,
// 					Angle: 0,
// 				}).
// 				Do(ctx)
// 			if err != nil {
// 				return errors.Wrapf(err, "failed to force viewport emulation")
// 			}

// 			return nil
// 		}),
// 	)
// 	return
// }

// func captureScreen(ctx context.Context, filename string, quality int) (e error) {
// 	var contentSize *dom.Rect
// 	if contentSize, e = forceViewportEmulation(ctx); e != nil {
// 		return
// 	}

// 	buf := make([]byte, 0, 1024)
// 	bufptr := &buf
// 	// capture entire browser viewport, returning png with quality=90
// 	if e = chromedp.Run(ctx,
// 		chromedp.ActionFunc(func(ctx context.Context) (err error) {
// 			// capture screenshot
// 			*bufptr, err = page.CaptureScreenshot().
// 				WithQuality(int64(quality)).
// 				WithClip(&page.Viewport{
// 					X:      contentSize.X,
// 					Y:      contentSize.Y,
// 					Width:  contentSize.Width,
// 					Height: contentSize.Height,
// 					Scale:  1,
// 				}).Do(ctx)
// 			if err != nil {
// 				return errors.Wrapf(err, "failed to capture screenshot for the page")
// 			}
// 			return nil
// 		}),
// 	); e != nil {
// 		return errors.Wrapf(e, "failed to capture screen")
// 	}
// 	imgPath := filepath.Join(
// 		conf.Args.WebDriver.WorkingFolder,
// 		fmt.Sprintf("%s_%s.png", filename, time.Now().Format("20060102_150405")),
// 	)
// 	if e = ioutil.WriteFile(
// 		imgPath,
// 		buf,
// 		0644); e != nil {
// 		return errors.Wrapf(e, "unable to save image to %s", imgPath)
// 	}

// 	return
// }
