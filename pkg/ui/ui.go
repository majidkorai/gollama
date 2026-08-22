package ui

import (
	_ "embed"
	"strings"
)

//go:embed logo.svg
var LogoSVG string

//go:embed web/index.html
var pageTemplate string

//go:embed web/app.css
var pageCSS string

//go:embed web/app.js
var pageJS string

// Page is the full HTML document served at "/". It is assembled from the
// embedded web/ files (P5-T4): the HTML template with the CSS and JS inlined
// at the __GOLLAMA_CSS__ / __GOLLAMA_JS__ placeholders. The served page stays
// byte-identical to the old single const (see TestPageMatchesReference),
// while HTML/CSS/JS now live in real, separately-editable files.
var Page = strings.NewReplacer(
	"__GOLLAMA_CSS__", pageCSS,
	"__GOLLAMA_JS__", pageJS,
).Replace(pageTemplate)
