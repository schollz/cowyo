package octicon_test

import (
	"io"
	"log"
	"os"

	"github.com/shurcooL/octicon"
	"golang.org/x/net/html"
)

func Example() {
	var w io.Writer = os.Stdout // Or, e.g., http.ResponseWriter in your HTTP handler, etc.

	err := html.Render(w, octicon.Alert())
	if err != nil {
		log.Fatalln(err)
	}

	// Output:
	// <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" style="fill: currentColor; vertical-align: top;"><path d="M8.893 1.5c-.183-.31-.52-.5-.887-.5s-.703.19-.886.5L.138 13.499a.98.98 0 0 0 0 1.001c.193.31.53.501.886.501h13.964c.367 0 .704-.19.877-.5a1.03 1.03 0 0 0 .01-1.002L8.893 1.5zm.133 11.497H6.987v-2.003h2.039v2.003zm0-3.004H6.987V5.987h2.039v4.006z"></path></svg>
}

func ExampleSetSize() {
	var w io.Writer = os.Stdout // Or, e.g., http.ResponseWriter in your HTTP handler, etc.

	err := html.Render(w, octicon.SetSize(octicon.MarkGitHub(), 24))
	if err != nil {
		log.Fatalln(err)
	}

	// Output:
	// <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 16 16" style="fill: currentColor; vertical-align: top;"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0 0 16 8c0-4.42-3.58-8-8-8z"></path></svg>
}
