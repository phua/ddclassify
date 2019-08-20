package main

import (
    "encoding/json"
    "encoding/xml"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
)

var verbose bool

var (
    nsfaRegexp = regexp.MustCompile(`\d+(?:\.\d*)?`)
    titleRegexp = regexp.MustCompile(`(?P<title>[^:]+)(?: : )(?P<subtitle>.*)?`)
    titleTemplate = `${title}_ ${subtitle}`
    authorRegexp = regexp.MustCompile(`(?P<surname>[^,|]+), (?P<prename>[^(,\[|]+ ?[^ (,\[|]+?)`)
    authorTemplate = `${prename} ${surname}`
)

type DDC struct {
    XMLName xml.Name `xml:"ddc"`
    Classes []*Class `xml:"class"`
}

type Class struct {
    XMLName xml.Name `xml:"class"`
    Number string `xml:"number,attr"`
    Description string `xml:"description,attr"`
    Divisions []*Division`xml:"division"`
}

type Division struct {
    XMLName xml.Name `xml:"division"`
    Number string `xml:"number,attr"`
    Description string `xml:"description,attr"`
    Sections []*Section `xml:"section"`
}

type Section struct {
    XMLName xml.Name `xml:"section"`
    Number string `xml:"number,attr"`
    Description string `xml:"description,attr"`
}

func parseDDC(xmlfile string) (DDC, error) {
    var ( ddc DDC ; err error ; contents []byte )
    if contents, err = ioutil.ReadFile(xmlfile); err == nil {
        err = xml.Unmarshal(contents, &ddc)
    }
    return ddc, err
}

type Classify struct {
    XMLName xml.Name `xml:"classify"`
    Response Response `xml:"response"`
    Input Input `xml:"input"`
    Works []Work `xml:"works>work"`
    Work Work `xml:"work"`
    Authors []Author `xml:"authors>author"`
    MostPopulars []MostPopular `xml:"recommendations>ddc>mostPopular"`
    MostPopular MostPopular
}

type Response struct {
    Code string `xml:"code,attr"`
}

type Input struct {
    Type string `xml:"type,attr"`
    Value string `xml:",chardata"`
}

type Work struct {
    Author string `xml:"author,attr"`
    Owi string `xml:"owi,attr"`
    Schemes string `xml:"schemes,attr"`
    Title string `xml:"title,attr"`
    Wi string `xml:"wi,attr"`
}

type Author struct {
    Value string `xml:",chardata"`
}

type MostPopular struct {
    Nsfa string `xml:"nsfa,attr"`
    Sfa string `xml:"sfa,attr"`
}

func parseClassify(body []byte) Classify {
    var ( classify Classify ; err error )
    if err = xml.Unmarshal(body, &classify); err != nil {
        classify.Response.Code = "999"
    }
    switch classify.Response.Code {
    case "0":                   // Single-work summary
        fallthrough
    case "2":                   // Single-work detail
        for _, mostPopular := range classify.MostPopulars {
            if nsfaRegexp.MatchString(mostPopular.Nsfa) {
                classify.MostPopular = mostPopular
            }
        }
        return classify
    case "4":                   // Multi-work
        for _, work := range classify.Works {
            if strings.Contains(work.Schemes, "DDC") && classify.Input.Type != "owi" {
                if c := sendRequestOCLC_OWI(work.Owi); c.MostPopular.Nsfa != "" {
                    return c
                }
            }
        }
    case "100":
        log.Println("No input")
    case "101":
        log.Println("Invalid input")
    case "102":
        log.Println("Not found")
    case "200":
        log.Println("Unexpected error")
    default:
        log.Println("Unknown error", err)
    }
    return classify
}

type Volume struct {
    TotalItems int
    Items []Item
}

type Item struct {
    VolumeInfo VolumeInfo
}

type VolumeInfo struct {
    Title string
    Authors []string
    IndustryIdentifiers []IndustryIdentifier
}

type IndustryIdentifier struct {
    Type string
    Identifier string
}

func parseVolume(body []byte) Volume {
    var volume Volume
    if err := json.Unmarshal(body, &volume); err != nil {
        log.Println(err)
    }
    return volume
}

func sendRequest(request string) (body []byte, err error) {
    log.Println("GET", request)
    response, err := http.Get(request)
    if err != nil {
        return
    }
    defer response.Body.Close()
    body, err = ioutil.ReadAll(response.Body)
    if err != nil {
        return
    }
    if verbose {
        log.Println(string(body))
    }
    return
}

func sendRequestOCLC(request string) Classify {
    if response, err := sendRequest(request); err != nil {
        log.Println(err)
        return Classify {}
    } else {
        return parseClassify(response)
    }
}

func sendRequestOCLC_TitleAuthor(title string, author string) Classify {
    request := "http://classify.oclc.org/classify2/Classify?title=%s&author=%s&summary=true"
    if index := strings.Index(author, ","); index > 0 { // TODO
        author = author[:index]
    }
    return sendRequestOCLC(fmt.Sprintf(request, url.PathEscape(title), url.PathEscape(author)))
}

func sendRequestOCLC_OWI(owi string) Classify {
    request := "http://classify.oclc.org/classify2/Classify?owi=%s&summary=true"
    return sendRequestOCLC(fmt.Sprintf(request, owi))
}

func sendRequestOCLC_ISBN(isbn string) Classify {
    request := "http://classify.oclc.org/classify2/Classify?isbn=%s&summary=true"
    return sendRequestOCLC(fmt.Sprintf(request, isbn))
}

func sendRequestGoogle(title string, author string) Classify {
    request := "https://www.googleapis.com/books/v1/volumes?q=+intitle:%s+inauthor:%s"
    response, err := sendRequest(fmt.Sprintf(request, url.PathEscape(title), url.PathEscape(author)))
    if err != nil {
        log.Println(err)
        return Classify {}
    }
    var classify Classify
    if volume := parseVolume(response); volume.TotalItems > 0 {
        for _, item := range volume.Items {
            for _, industryIdentifier := range item.VolumeInfo.IndustryIdentifiers {
                if strings.HasPrefix(industryIdentifier.Type, "ISBN") {
                    classify = sendRequestOCLC_ISBN(industryIdentifier.Identifier)
                    if classify.MostPopular.Nsfa != "" {
                        return classify
                    }
                }
            }
        }
    }
    return classify
}

func classify(classify Classify, ddc DDC, depth int) string {
    if classify.MostPopular.Nsfa == "" {
    } else if i, err := strconv.Atoi(classify.MostPopular.Nsfa[:3]); err != nil {
        log.Println(err)
    } else if class := ddc.Classes[i / 100]; depth == 1 {
        return filepath.Join(class.Description)
    } else if division := class.Divisions[(i / 10) % 10]; depth == 2 {
        return filepath.Join(class.Description, division.Description)
    } else {
        section := division.Sections[i % 10]
        return filepath.Join(class.Description, division.Description, section.Description)
    }
    return "Unassigned"
}

func capture(s string, re *regexp.Regexp, t string) string {
    if re.MatchString(s) {
        b := []byte {}
        for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
            b = re.ExpandString(b, t, s, m)
        }
        return string(b)
    }
    return s
}

func titleString(title string) string {
    return strings.Title(capture(title, titleRegexp, titleTemplate))
}

func authorsString(authors []Author) string {
    b, n := strings.Builder {}, len(authors) - 1
    for i, author := range authors {
        b.WriteString(capture(author.Value, authorRegexp, authorTemplate))
        if i < n {
            b.WriteString(", ")
        }
    }
    return b.String()
}

func classifyString(c Classify, ddc DDC, depth int) string {
    return fmt.Sprintf("%s : %s/%s - %s", c.MostPopular.Nsfa, classify(c, ddc, depth),
        titleString(c.Work.Title), authorsString(c.Authors))
}

func classifyFilename(c Classify, ext string) string {
    return fmt.Sprintf("%s - %s%s", titleString(c.Work.Title), authorsString(c.Authors), ext)
}

func parseFilenameRegexp(name string, re *regexp.Regexp) (title string, author string) {
    if re.MatchString(name) {
        split := strings.Split(capture(name, re, "${title}|${author}"), "|")
        title, author = split[0], split[1]
    }
    return
}

func scan(path string, recurse bool, excludes []string,
    parseFilename func(string) (string, string),
    sendRequest func(string, string) Classify,
    processResponse func(string, Classify)) {
    filepath.Walk(path,
        func(_path string, fileInfo os.FileInfo, err error) error {
            if err != nil {
                log.Fatal(err)
            }
            if fileInfo.IsDir() {
                if recurse {
                    for _, exclude := range excludes {
                        if exclude != "" && strings.Contains(_path, exclude) {
                            return filepath.SkipDir
                        }
                    }
                } else if filepath.Base(path) != fileInfo.Name() {
                    return filepath.SkipDir
                }
            } else {
                var classify Classify // "Unassigned"
                if title, author := parseFilename(fileInfo.Name()); title != "" {
                    classify = sendRequest(title, author)
                }
                processResponse(_path, classify)
            }
            return nil
        })
}

const (
    COPY = 1
    LINK = 2
    SLNK = 4
    MOVE = 8
    RNME = 16
)

func copyFile(from string, to string) error {
    src, err := os.Open(from)
    if err != nil {
        return err
    }
    defer src.Close()

    dst, err := os.Create(to)
    if err != nil {
        return err
    }
    defer dst.Close()

    if _, err := io.Copy(dst, src); err != nil {
        return err
    }
    return dst.Sync()
}

func linkFile(src string, dst string, mode int) (err error) {
    log.Println("mkdir", filepath.Dir(dst))
    if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
        return
    }
    switch mode & 0xF {
    case COPY:
        log.Println("cp", src, dst)
        return copyFile(src, dst)
    case LINK:
        log.Println("ln", src, dst)
        return os.Link(src, dst)
    case SLNK:
        log.Println("ln -s", src, dst)
        return os.Symlink(src, dst)
    case MOVE:
        log.Println("mv", src, dst)
        return os.Rename(src, dst)
    default:
        log.Println("ln", src, dst)
    }
    return
}

func main() {
    title := flag.String("t", "", "Search by title.")
    author := flag.String("a", "", "Search by author and title.")
    isbn := flag.String("i", "", "Search by ISBN.")
    directory := flag.String("d", ".", "Search directory.")
    pattern := flag.String("p", `^(?P<title>.+?)(,.*Edition)? - (?P<author>.+)\.([A-Za-z]+)$`, "Regular expression pattern for filenames.")
    recurse := flag.Bool("r", false, "Search directory recursively.")
    exclude := flag.String("e", "", "Exclude search directories.")
    create := flag.String("c", "", "Create DDC directory structure and transfer files.")
    mode := flag.Int("m", 0, "File transfer mode: copy(1), link(2), symlink(4), move(8).")
    xml := flag.String("x", "ddc.xml", "DDC definition XML file.")
    depth := flag.Int("depth", 3, "Classification depth: class(1), division(2), section(3).")
    google := flag.Bool("g", false, "Use the Google Books API to lookup the ISBN.")
    flag.BoolVar(&verbose, "v", false, "Verbose. Log HTTP responses.")
    flag.Parse()
    if flag.NFlag() < 1 {
        fmt.Println("Usage:\n\t$", os.Args[0], "[-t title [-a author]] [-i isbn] [-d path [-p pattern] [-r] [-e directories] [-c /path/to/library [-m 1|2|4|8]]] [-x /path/to/ddc.xml] [-depth 1..3] [-g] [-v]")
        return
    }
    ddc, err := parseDDC(*xml)
    if err != nil {
        log.Println(err)
        return
    }
    sendRequest := sendRequestOCLC_TitleAuthor
    if *google {
        sendRequest = sendRequestGoogle
    }
    flag.Visit(func(flag *flag.Flag) { // lexicographical order
        switch flag.Name {
        case "t":
            fmt.Println(classifyString(sendRequest(*title, *author), ddc, *depth))
        case "i":
            fmt.Println(classifyString(sendRequestOCLC_ISBN(*isbn), ddc, *depth))
        case "d":
            excludes := strings.Split(*exclude, ",")
            filenameRegexp := regexp.MustCompile(*pattern)
            parseFilename := func(name string) (string, string) {
                return parseFilenameRegexp(name, filenameRegexp)
            }
            processResponse := func(path string, c Classify) {
                fmt.Println(classifyString(c, ddc, *depth))
            }
            if *create != "" {
                processResponse = func(path string, c Classify) {
                    name := filepath.Base(path)
                    if *mode & RNME == RNME && c.MostPopular.Nsfa != "" {
                        name = classifyFilename(c, filepath.Ext(path))
                    }
                    err := linkFile(path, filepath.Join(*create, classify(c, ddc, *depth), name), *mode)
                    if err != nil {
                        log.Println(err)
                    }
                }
            }
            scan(*directory, *recurse, excludes, parseFilename, sendRequest, processResponse)
        }
    })
}
