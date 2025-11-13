//go:build go1.19

package dials

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	statuspage "github.com/vimeo/go-status-page"
)

// Dials is the main access point for your configuration.
type Dials[T any] struct {
	value       atomic.Pointer[versionedConfig[T]]
	updatesChan chan *T
	params      Params[T]
	cbch        chan<- userCallbackEvent
	monCtl      chan<- verifyEnable[T]
	dumpStack   chan<- dumpSourceStack[T]

	// sourceVals and defVal are only present if the status page is enabled
	// and there are no watching sources. (or the monitor has exited)
	sourceVals atomic.Pointer[[]sourceValue]
	defVal     *T
}

// View returns the configuration struct populated.
func (d *Dials[T]) View() *T {
	versioned := d.value.Load()
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return versioned.cfg
}

// ViewVersion returns the configuration struct populated, and an opaque token.
func (d *Dials[T]) ViewVersion() (*T, CfgSerial[T]) {
	versioned := d.value.Load()
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return versioned.cfg, CfgSerial[T]{s: versioned.serial, cfg: versioned.cfg}
}

// ServeHTTP is only active if [Params.EnableStatusPage] was set to true when creating the dials instance.
//
// This is experimental and may panic while serving, buyer beware!!!
//
// It is heavily recommended that users of this functionality set
// `statuspage:"-"` tags on any sensitive fields with secrets/credentials, etc.
func (d *Dials[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !d.params.EnableStatusPage {
		http.Error(w, "Dials status page not enabled. EnableStatusPage must be set in dials.Params.", http.StatusNotFound)
		return
	}
	srcs := d.sourceVals.Load()
	_, serial := d.ViewVersion()
	if srcs == nil {
		// Make the response channel size-1 so the monitor doesn't block on the response
		respCh := make(chan dumpSourceStackResponse[T], 1)
		// ask the monitor for the stack
		select {
		case d.dumpStack <- dumpSourceStack[T]{resp: respCh}:
		case <-r.Context().Done():
			// not worth trying to resolve the race around when the monitor shuts down for a status page.
			// just return a 500.
			http.Error(w, "context expired while attempting to request the current source-stack; please try again.",
				http.StatusInternalServerError)
			return
		}
		select {
		case v := <-respCh:
			srcs = &v.stack
			serial = v.serial
		case <-r.Context().Done():
			// not worth trying to resolve the race around when the monitor shuts down for a status page.
			// just return a 500.
			http.Error(w, "context expired while attempting to acquire the current stacks; please try again.",
				http.StatusInternalServerError)
			return
		}
	}

	root := html.Node{Type: html.DocumentNode}
	root.AppendChild(&html.Node{
		Type:     html.DoctypeNode,
		DataAtom: atom.Html,
		Data:     atom.Html.String(),
	})
	htmlElem := createElemAtom(atom.Html)
	root.AppendChild(htmlElem)

	head := createElemAtom(atom.Head)

	htmlElem.AppendChild(head)
	title := createElemAtom(atom.Title)
	title.AppendChild(textNode("Dials Status"))
	head.AppendChild(title)
	header := createElemAtom(atom.H1)
	header.AppendChild(textNode("Dials Status"))
	head.AppendChild(header)

	body := createElemAtom(atom.Body)
	htmlElem.AppendChild(body)

	curCfg, genCfgErr := statuspage.GenHTMLNodes(serial.cfg)
	if genCfgErr != nil {
		http.Error(w, fmt.Sprintf("failed to render status page for current config of type %T: %s.", serial.cfg, genCfgErr),
			http.StatusInternalServerError)
		return
	}
	curStatusH2 := createElemAtom(atom.H2)
	curStatusH2.AppendChild(textNode("current configuration"))
	body.AppendChild(curStatusH2)
	curStatusVers := createElemAtom(atom.P)
	curStatusVers.AppendChild(textNode(fmt.Sprintf("Current Serial: %d", serial.s)))
	body.AppendChild(curStatusVers)

	for _, cfgNode := range curCfg {
		// add a horizontal rule to separate sections
		body.AppendChild(createElemAtom(atom.Hr))
		body.AppendChild(cfgNode)
	}
	defCfgH2 := createElemAtom(atom.H2)
	defCfgH2.AppendChild(textNode("Default Configuration"))
	body.AppendChild(defCfgH2)
	defCfgNodes, defCfgErr := statuspage.GenHTMLNodes(d.defVal)
	if defCfgErr != nil {
		http.Error(w, fmt.Sprintf("failed to render status page for default config of type %T: %s.",
			serial.cfg, defCfgErr),
			http.StatusInternalServerError)
		return
	}

	for _, cfgNode := range defCfgNodes {
		// add a horizontal rule to separate sections
		body.AppendChild(createElemAtom(atom.Hr))
		body.AppendChild(cfgNode)
	}

	for srcIdx, srcVal := range *srcs {
		body.AppendChild(createElemAtom(atom.Hr))
		srcSectionHeader := createElemAtom(atom.H2)
		srcSectionHeader.AppendChild(textNode(fmt.Sprintf("Source %d of type %T (watching %t)", srcIdx, srcVal.source, srcVal.watching)))
		body.AppendChild(srcSectionHeader)
		srcBodyNodes, srcBodyGenErr := statuspage.GenHTMLNodes(srcVal.value.Interface())
		if srcBodyGenErr != nil {
			http.Error(w, fmt.Sprintf("failed to render status page for config of type %T on source %d: %s.", serial.cfg, srcIdx, srcBodyGenErr),
				http.StatusInternalServerError)
			return
		}
		for _, bn := range srcBodyNodes {
			// add a horizontal rule to separate sections
			body.AppendChild(createElemAtom(atom.Hr))
			body.AppendChild(bn)
		}
	}

	if renderErr := html.Render(w, htmlElem); renderErr != nil {
		http.Error(w, fmt.Sprintf("failed to render status page into html for config of type %T : %s.", serial.cfg, renderErr),
			http.StatusInternalServerError)
	}
}

func createElemAtom(d atom.Atom) *html.Node {
	n := &html.Node{Type: html.ElementNode, DataAtom: d, Data: d.String()}
	if n.DataAtom == atom.Table {
		n.Attr = append(n.Attr, html.Attribute{
			Key: "style",
			Val: "border: 1px solid; min-width: 100px",
		})
	}
	return n
}

func textNode(d string) *html.Node {
	return &html.Node{Type: html.TextNode, Data: d}
}
