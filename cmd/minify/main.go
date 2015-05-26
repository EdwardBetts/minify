package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

func main() {
	input := ""
	output := ""
	ext := ""
	directory := ""
	recursive := false

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [file]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&output, "o", "", "Output file (stdout when empty)")
	flag.StringVar(&ext, "x", "", "File extension (css, html, js, json, svg or xml), optional for input files")
	flag.StringVar(&directory, "d", "", "Directory to search for files")
	flag.BoolVar(&recursive, "r", false, "Recursively minify everything")
	flag.Parse()
	if len(flag.Args()) > 0 {
		input = flag.Arg(0)
	}

	extPassed := (ext != "")

	mediatype := ""
	r := io.Reader(os.Stdin)
	w := io.Writer(os.Stdout)
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	filenames := make(map[string]string)
	if directory != "" {
		filenames = ioNames(directory, recursive)
	} else {
		filenames[input] = output
	}

	for in, out := range filenames {
		input = in
		output = out

		if input != "" {
			in, err := os.Open(input)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			defer in.Close()
			r = in
			if input == output {
				b := &bytes.Buffer{}
				io.Copy(b, r)
				r = b
			}
			if !extPassed {
				ext = filepath.Ext(input)
			}
		}
		if output != "" {
			out, err := os.Create(output)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			defer out.Close()
			w = out
		}
		if ext != "" {
			switch ext {
			case ".css":
				mediatype = "text/css"
			case ".html":
				mediatype = "text/html"
			case ".js":
				mediatype = "application/javascript"
			case ".json":
				mediatype = "application/json"
			case ".svg":
				mediatype = "image/svg+xml"
			case ".xml":
				mediatype = "text/xml"
			}
		}

		if err := m.Minify(mediatype, w, r); err != nil {
			if err == minify.ErrNotExist {
				io.Copy(w, r)
			} else {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
		}
	}
}

// ioNames returns a map of input paths and output paths.
func ioNames(startDir string, recursive bool) map[string]string {
	names := make(map[string]string)

	if recursive {
		filepath.Walk(startDir, func(path string, info os.FileInfo, _ error) error {
			if !validFile(info) {
				return nil
			}

			names[path] = minExt(path)

			return nil
		})

		return names
	}

	infos, err := ioutil.ReadDir(startDir)
	if err != nil {
		return map[string]string{}
	}

	for _, info := range infos {
		if !validFile(info) {
			continue
		}

		fullPath := filepath.Join(startDir, info.Name())
		names[fullPath] = minExt(fullPath)
	}

	return names
}

// validFile checks to see if a file is a directory, hidden, already has the
// minified extension, or if it's one of the minifiable extensions.
func validFile(info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	if info.Name()[0] == '.' {
		return false
	}

	// don't want to reminify already minified files
	if strings.Contains(info.Name(), ".min.") {
		return false
	}

	switch strings.ToLower(filepath.Ext(info.Name())) {
	case ".css", ".html", ".js", ".json", ".svg", ".xml":
		return true
	default:
		return false
	}
}

// minExt adds .min before a file's extension.
func minExt(path string) string {
	var dot int

	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			dot = i
			break
		}
	}

	return path[:dot] + ".min" + filepath.Ext(path)
}
