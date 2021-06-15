package browser

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/logging"
	"github.com/agux/roprox/types"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

var log = logging.Logger

type Chrome struct {
	ctx context.Context
	cf  context.CancelFunc
}

func (c *Chrome) Cancel() {
	c.cf()
}

func (c *Chrome) Run(timeout time.Duration, actions ...chromedp.Action) (e error) {
	ctx, cf := context.WithTimeout(c.ctx, timeout)
	defer cf()
	return chromedp.Run(ctx, actions...)
}

func LaunchChrome(url, userAgent string, proxy *types.ProxyServer, timeout time.Duration) (c *Chrome, e error) {
	var o []chromedp.ExecAllocatorOption
	// prepare options
	if proxy != nil {
		p := fmt.Sprintf("%s://%s:%s", proxy.Type, proxy.Host, proxy.Port)
		log.Debugf("chrome is using proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	}
	if len(userAgent) > 0 {
		log.Debugf("chrome is using user agent: %s", userAgent)
		o = append(o, chromedp.UserAgent(userAgent))
	}
	if conf.Args.WebDriver.NoImage {
		o = append(o, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}
	// o = append(o, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)

	for _, opt := range chromedp.DefaultExecAllocatorOptions {
		if reflect.ValueOf(chromedp.Headless).Pointer() == reflect.ValueOf(opt).Pointer() {
			if conf.Args.WebDriver.Headless {
				log.Debug("headless mode is enabled")
			} else {
				log.Debug("ignored headless mode")
				continue
			}
		}
		o = append(o, opt)
	}

	//start
	ctx, _ := chromedp.NewExecAllocator(
		context.Background(),
		o...)
	ctx, cf := chromedp.NewContext(ctx)

	c = &Chrome{
		ctx, cf,
	}

	//head to given URL if provided
	if len(url) > 0 {
		tm := time.AfterFunc(timeout, cf)
		if e = chromedp.Run(ctx, chromedp.Navigate(url)); e != nil {
			defer cf()
			//TODO maybe it will not timeout when using a bad proxy, and shows chrome error page instead
			e = errors.Wrapf(e, "failed to navigate %s", url)
			log.Error(e)
			return
		}
		tm.Stop()
	}
	return
}
