package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

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