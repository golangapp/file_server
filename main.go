package main

import (
	"flag"
	"fmt"
	// "github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"path/filepath"
	"strings"
	tt "text/template"
)

var (
	dir                 string
	port                string
	logging             bool
	depth               int
	auth                string
	debug               bool
	disable_sys_command bool
	rootDir             = "."
)

//var cpuprof string
//commandsFile        string

const MAX_MEMORY = 1 * 1024 * 1024
const VERSION = "1.1"

func main() {

	//fmt.Println(len(os.Args), os.Args)
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("Version " + VERSION)
		os.Exit(0)
	}

	flag.StringVar(&dir, "dir", ".", "Specify a directory to server files from.")
	flag.StringVar(&port, "port", ":8080", "Port to bind the file server")
	flag.BoolVar(&logging, "log", true, "Enable Log (true/false)")
	flag.StringVar(&auth, "auth", "", "'username:pass' Basic Auth")
	flag.IntVar(&depth, "depth", 5, "Depth directory crawler")
	//flag.StringVar(&commandsFile, "commands", "", "Path to external commands file.json")
	flag.BoolVar(&debug, "debug", false, "Make external assets expire every request")
	flag.BoolVar(&disable_sys_command, "disable_cmd", true, "Disable sys comands")

	//flag.StringVar(&cpuprof, "cpuprof", "", "write cpu and mem profile")
	flag.Parse()

	envDir := os.Getenv("FILESERVER_DIR")
	if envDir != "" {
		dir = envDir
	}
	envPort := os.Getenv("FILESERVER_PORT")
	if envPort != "" {
		port = envPort
	}
	envAuth := os.Getenv("FILESERVER_AUTH")
	if envAuth != "" {
		auth = envAuth
	}
	envCmd := os.Getenv("FILESERVER_COMMAND")
	if envCmd != "" {
		disable_sys_command = false
	}

	if logging == false {
		log.SetOutput(ioutil.Discard)
	}
	// If no path is passed to app, normalize to path formath
	if dir == "." {
		dir, _ = filepath.Abs(dir)
	}

	if _, err := os.Stat(dir); err != nil {
		log.Fatalf("Directory %s not exist", dir)
	}

	// normalize dir, ending with... /
	if strings.HasSuffix(dir, "/") == false {
		dir = dir + "/"
	}

	// build index files in background
	go Build_index(dir)

	mux := http.NewServeMux()

	statics := &ServeStaticFromBinary{
		MountPoint: "/-/assets/",
		DataDir:    "data/"}

	mux.Handle("/-/assets/", makeGzipHandler(statics.ServeHTTP))

	mux.Handle("/-/api/dirs", makeGzipHandler(http.HandlerFunc(SearchHandle)))
	mux.Handle("/", BasicAuth(http.HandlerFunc(handleReq), auth))

	log.Printf("Listening on port %s .....\n", port)
	if debug {
		log.Print("Serving data dir in debug mode.. no assets caching.\n")
	}
	http.ListenAndServe(port, mux)

}

func handleReq(w http.ResponseWriter, r *http.Request) {

	//Is_Ajax := strings.Contains(r.Header.Get("Accept"), "application/json")
	if r.Method == "PUT" {
		AjaxUpload(w, r)
		return
	}
	if r.Method == "POST" {
		WebCommandHandler(w, r)
		return
	}

	log.Print("Request: ", r.RequestURI)
	// See bug #9. For some reason, don't arrive index.html, when asked it..
	if strings.HasSuffix(r.URL.Path, "/") && r.FormValue("get_file") != "true" {
		log.Printf("Index dir %s", r.URL.Path)
		handleDir(w, r)
	} else {
		cleanPath := path.Clean(dir + r.URL.Path)
		log.Printf("downloading file %s", cleanPath)
		r.Header.Del("If-Modified-Since")
		if strings.Contains(cleanPath, ".md") && r.FormValue("get_file") != "true" {
			htmlFlags := 0
			htmlFlags |= blackfriday.HTML_FOOTNOTE_RETURN_LINKS

			// set up options
			extensions := 0
			extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
			extensions |= blackfriday.EXTENSION_TABLES
			extensions |= blackfriday.EXTENSION_FENCED_CODE
			extensions |= blackfriday.EXTENSION_AUTOLINK
			extensions |= blackfriday.EXTENSION_STRIKETHROUGH
			extensions |= blackfriday.EXTENSION_SPACE_HEADERS
			// extensions |= blackfriday.EXTENSION_HARD_LINE_BREAK

			fi, _ := os.Stat(cleanPath)
			css := "/-/assets/github.css"
			title := fi.Name()
			input, _ := ioutil.ReadFile(cleanPath)

			renderer := blackfriday.HtmlRenderer(htmlFlags, title, css)

			output := blackfriday.Markdown(input, renderer, extensions)

			tmpl, _ := tt.New("tmpl").Parse(templContent)
			md := mdContent{Content: string(output)}
			tmpl.Execute(w, md)
			// bc := blackfriday.MarkdownCommon(c)
			// html := bluemonday.UGCPolicy().SanitizeBytes(bc)
			// fmt.Fprintf(w, "%s", output)
		} else {
			http.ServeFile(w, r, cleanPath)
		}
		//http.ServeContent(w, r, r.URL.Path)
		//w.Write([]byte("this is a test inside file handler"))

	}

}

func handleDir(w http.ResponseWriter, r *http.Request) {

	var d string = ""

	//log.Printf("len %d,, %s", len(r.URL.Path), dir)
	if len(r.URL.Path) == 1 {
		// handle root dir
		d = dir
	} else {
		d += dir + r.URL.Path[1:]
	}

	// handle json format of dir...
	if r.FormValue("format") == "json" {

		w.Header().Set("Content-Type", "application/json")
		result := &DirJson{w, d}
		err := result.Get()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if r.FormValue("format") == "zip" {
		result := &DirZip{w, d}
		err := result.Get()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// If we dont receive json param... we are asking, for genric app ui...
	template_file, err := Asset("data/main.html")
	if err != nil {
		log.Fatalf("Cant load template main")
	}

	t := template.Must(template.New("listing").Delims("[%", "%]").Parse(string(template_file)))
	v := map[string]interface{}{
		"Path":        r.URL.Path,
		"version":     VERSION,
		"sys_command": disable_sys_command,
	}
	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, v)

}

func AjaxUpload(w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()
	if err != nil {
		fmt.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pa := r.URL.Path[1:]

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		var ff string
		if dir != "." {
			ff = dir + pa + part.FileName()
		} else {
			ff = pa + part.FileName()
		}

		dst, err := os.Create(ff)
		defer dst.Close()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, part); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprint(w, "ok")
	return
}
