/*
 * Kyle Thomas
 * Most of this is "borrowed" from https://schier.co/blog/2015/04/26/a-simple-web-scraper-in-go.html
 * and from https://golangcode.com/download-a-file-from-a-url/
 * Small application to download files listed on a webserver
 * Usage: $ grabfiles url [extenstions]
 *        where extenstions are an optional list of file extenstions to grab
 *        default ones are ".c" ".h" ".pdf"
 */


package main


import (
    "fmt"
    "io"
    "golang.org/x/net/html"
    "os"
    "net/http"
    "strings"
)


// Pull the href attribute from token
func getHref(t html.Token) (ok bool, href string) {
    for _, a := range t.Attr {
        if a.Key == "href" {
            href = a.Val
            ok = true
        }
    }
    return
}

// Extract links with matching extensions
func crawl(url string, extenstions []string) []string {
    var files []string

    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
        return nil
    }

    html_body := resp.Body
    defer html_body.Close()  // close Body when function returns

    tokens := html.NewTokenizer(html_body)

    for {
        tt := tokens.Next()

        switch {
        case tt == html.ErrorToken:
            // End of doc
            return files
        case tt == html.StartTagToken:
            token := tokens.Token()

            // Check if token is an <a> tag
            isAnchor := token.Data == "a"
            if !isAnchor {
                continue
            }

            // Extract href value if there is one
            ok, url := getHref(token)
            if !ok {
                continue
            }

            // Make sure the url has wanted file type
            for _, extenstion := range extenstions {
                if strings.HasSuffix(url, extenstion) {
                    files = append(files, url)
                }
            }
        }
    }
    return files
}


// download the file from the url onto local drive with same name
func downloadFile(urlpath, file string, ch chan bool, chFinished chan bool) {
    defer func() {  // Signal that download is done
        chFinished <- true
    }()

    // get data
    resp, err := http.Get(urlpath + file)
    if err != nil {
        fmt.Println(" - ERROR\t", urlpath, file)
        ch <- false
        return
    }
    defer resp.Body.Close()

    // create file
    out, err := os.Create(file)
    if err != nil {
        fmt.Println(" - ERROR\t", file)
        ch <- false
        return
    }
    defer out.Close()

    // write contents to file
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        fmt.Println(" - ERROR\t", file)
        ch <- false
        return
    }

    fmt.Println(" - SUCCESS\t", file)
    ch <- true
    return
}


// print out usage with -h argument or too few arguments
func usage() {
    fmt.Println("Usage:")
    fmt.Println("\t$ grabfiles url [extenstions to download]")
    fmt.Println("\tWhere extensions are option, default ones are \".c\", \".h\", and \".pdf\"")
}


func main() {
    if (len(os.Args) < 2) || (os.Args[1] == "-h") {
        usage()
        os.Exit(1)
    }
    var extenstions []string
    if len(os.Args) == 2 {
        extenstions = append(extenstions, ".c", ".h", ".pdf")
    } else {
        extenstions = os.Args[2:]
    }

    files := crawl(os.Args[1], extenstions)
    if files == nil {
        os.Exit(1)
    }

    chDownloaded := make(chan bool)
    defer close(chDownloaded)
    chFinished := make(chan bool)
    defer close(chFinished)

    // download results concurrently
    fmt.Println("Attempting to download", len(files), "file(s)\n")
    var total_downloaded int
    for _, file := range files {
        go downloadFile(os.Args[1], file, chDownloaded, chFinished)
    }

    // subscribe to channels
    for c := 0; c < len(files); {
        select {
        case <- chFinished:
            c++
        case <- chDownloaded:
            total_downloaded++
        }
    }

    fmt.Println("")
    fmt.Println(total_downloaded, "file(s) successfully downloaded.\n")
    os.Exit(0)
}

