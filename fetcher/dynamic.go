package fetcher

import (
	"fmt"
	"strings"
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
					text.push(current.innerText + ",")
				}
				return text
			 };`

	invokeFuncJS := `var a = getText('` + sel + `'); a;`
	js = strings.Join([]string{funcJS, invokeFuncJS}, " ")
	log.Debugf("javascript: %s", js)
	return js
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
	log.Debugf("javascript: %s", js)
	return js
}
