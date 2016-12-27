package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ipfs/go-ipfs/core"
	"golang.org/x/net/context"
	//	"github.com/ipfs/go-ipfs/core/corenet"
	"encoding/json"
	"mime/multipart"

	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

func SetupIpfs() (*core.IpfsNode, error) {
	// Assume the user has run 'ipfs init'
	r, err := fsrepo.Open("~/.ipfs")
	if err != nil {
		fmt.Println(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &core.BuildCfg{
		Repo:   r,
		Online: true,
	}

	nd, err := core.NewNode(ctx, cfg)

	if err != nil {
		fmt.Println(err)
	}

	return nd, err
}

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() (string, error) {
	filename := p.Title + ".txt"
	fmt.Println("saving " + filename)

	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", filename)
	fw.Write(p.Body)
	if err != nil {
		fmt.Println("Error in save 1:", err)
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	req, err := http.NewRequest("POST", "http://localhost:5001/api/v0/add", &b)
	if err != nil {
		fmt.Println("Error in save 2:", err)
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error in save 3:", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("Resp body: " + string(body))

	var m map[string]string
	json.Unmarshal(body, &m)

	fmt.Println("Name: ", m["Name"])
	fmt.Println("Hash: ", m["Hash"])

	return m["Hash"], err
}

func loadPage(hash string) (*Page, error) {
	fmt.Println("Loading " + hash)
	path := "/ipfs/" + hash
	resp, err := http.Get("http://localhost:5001/api/v0/cat?arg=" + path)

	if err != nil {
		fmt.Println("Error in load: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("Resp body: " + string(body))
	return &Page{Title: hash, Body: body}, nil
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/edit/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	fmt.Fprintf(w, "<h1>Editing %s</h1>"+
		"<form action=\"/save/%s\" method=\"POST\">"+
		"<textarea name=\"body\">%s</textarea><br>"+
		"<input type=\"submit\" value=\"Save\">"+
		"</form>",
		p.Title, p.Title, p.Body)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/save/"):]
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}

	hash, err := p.save()
	if err != nil {
		fmt.Println("Error in save handler: ", err)
	}

	http.Redirect(w, r, "/view/"+hash, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/view/"):]
	p, _ := loadPage(title)
	fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

func main() {
	// nd, err := SetupIpfs()
	// if err != nil {
	//     fmt.Println(err)
	//     return
	// }

	// list, err := corenet.Listen(nd, "/app/x")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("I am peer: %s\n", nd.Identity.Pretty())

	// for {
	// 	con, err := list.Accept()
	// 	if err != nil {
	// 		fmt.Println("Error:", err)
	// 		return
	// 	}
	// 	defer con.Close()

	// 	fmt.Fprintln(con, "Hello! This is whyrusleepings awesome ipfs service")
	// 	fmt.Printf("Connection from: %s\n", con.Conn().RemotePeer())
	// }

	fmt.Println("Node setupped!")

	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	http.Handle("/", http.FileServer(http.Dir("html")))
	
	http.ListenAndServe(":8008", nil)
	
}
