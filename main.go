/*
Wercsrv is a simple go http server to use for serving werc-driven sites.

To use, run in a werc root or provide a "root" argument. The address
to serve HTTP on is set by and argument to the "addr" flag.

To set up werc, see the werc homepage: http://werc.cat-v.org

NOTES

Any port is stripped from the "Host" header passed to werc.

The PLAN9 environment variable is passed through to child processes,
but bin/werc.rc may need to be patched if plan9port is not in
/usr/local/plan9.
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

func main() {
	var (
		rootArg = flag.String("root", ".", "Root for werc tree.")
		addr    = flag.String("addr", ":8080", "Address to serve HTTP on.")
	)
	flag.Parse()
	root, err := filepath.Abs(*rootArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "!", err)
		return
	}
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	srv := &http.Server{
		Addr:    *addr,
		Handler: werc(root),
	}

	go func() {
		fmt.Fprintf(os.Stderr, "# using werc at %q\n", root)
		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintln(os.Stderr, "!", err)
		}
	}()

	<-sig
	if err := srv.Shutdown(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "!", err)
	}
	fmt.Fprintln(os.Stderr, "# exiting")
}

type wercHandler struct {
	root string
	werc *cgi.Handler
}

func (h *wercHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, p := range strings.Split(r.URL.Path, "/") {
		if p == "_werc" {
			http.NotFound(w, r)
			return
		}
	}
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		r.Host = h
	}
	fn := filepath.Join(h.root, "sites", r.Host, r.URL.Path)
	if fi, err := os.Stat(fn); err == nil && !fi.IsDir() {
		http.ServeFile(w, r, fn)
		return
	}
	h.werc.ServeHTTP(w, r)
}

func werc(root string) http.Handler {
	return &wercHandler{
		root: root,
		werc: &cgi.Handler{
			Path: filepath.Join(root, "bin", "werc.rc"),
			Dir:  filepath.Join(root, "bin"),
			InheritEnv: []string{
				"PLAN9",
			},
			PathLocationHandler: nil,
		},
	}
}
