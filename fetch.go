package gofetch

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	yaml "gopkg.in/yaml.v2"
)

// Config 规则信息
type Config struct {
	Key,
	Base string
	Index struct {
		URL      string
		Category *struct {
			Items string
		}
	}
	Login struct {
		URL        string
		PostURL    string `yaml:"postUrl"`
		CheckLogin string `yaml:"checkLogin"`
		Headers,
		Replace map[string][]string
		Convert map[string][][]string
	}
	Rules []*map[string]string
}

// ConfigRule 匹配URL对应的规则
type ConfigRule struct {
	Config  *Config
	Rule    *map[string]string
	IsIndex bool
}

// Res 返回的数据
type Res struct {
	Content map[string]string
	Categories,
	Items []map[string]string
}

// LoginInfo 登录信息
type LoginInfo struct {
	Username,
	Password,
	Captcha string
	Image    []byte
	ImageURL string
	Ext      map[string]string
}

// Fetch 数据获取实例
type Fetch struct {
	Config map[string]*Config
	Cookie map[string][]*http.Cookie
}

// New 创建数据获取实例
func New(configPaths ...string) (*Fetch, error) {
	fetch := &Fetch{
		make(map[string]*Config),
		make(map[string][]*http.Cookie),
	}
	// configPaths = append([]string{
	// 	"./rule/v2ex.yaml",
	// 	"./rule/hipda.yaml",
	// }, configPaths...)

	for _, configPath := range configPaths {
		config := &Config{}
		content, err := ioutil.ReadFile(configPath)
		if err != nil {
			log.Println(err)
			continue
		}

		err = yaml.Unmarshal(content, config)
		if err != nil {
			log.Println(err)
			continue
		}

		if config.Key != "" {
			fetch.Config[config.Key] = config
		}
	}

	return fetch, nil
}

// CreateLoginInfo 获取登录必须的数据
func (f *Fetch) CreateLoginInfo(key string) (*LoginInfo, error) {
	v := f.Config[key]
	if v != nil {
		dataURL := v.Base + v.Login.URL
		r, err := f.Data(dataURL)
		if err != nil {
			return nil, err
		}
		if r == nil {
			return nil, errors.New("no login form")
		}

		// mapDelAndGet := func(m map[string]string, k string) string {
		// 	v, ok := m[k]
		// 	if ok {
		// 		delete(m, k)
		// 	}
		// 	return v
		// }

		var image []byte
		imgURLKey := "captchaImgUrl"
		// iu := mapDelAndGet(r.Content, imgURLKey)
		iu, ok := r.Content[imgURLKey]
		if ok && iu != "" {
			delete(r.Content, imgURLKey)
			ri, ok := v.Login.Replace[imgURLKey]
			if ok && len(ri) > 0 {
				re := regexp.MustCompile(ri[0])
				iu = re.ReplaceAllString(iu, ri[1])
				link, err := url.Parse(iu)
				if err != nil {
					return nil, err
				}
				base, err := url.Parse(dataURL)
				if err != nil {
					return nil, err
				}
				iu = base.ResolveReference(link).String()
			}
			req, err := http.NewRequest("GET", iu, nil)
			if err != nil {
				return nil, err
			}
			cs := f.Cookie[v.Key]
			for _, c := range cs {
				req.AddCookie(c)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			f.Cookie[v.Key] = updateCookies(resp.Cookies(), cs)

			image, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
		}
		return &LoginInfo{
			// UsernameKey: mapDelAndGet(r.Content, "usernameKey"),
			// PasswordKey: mapDelAndGet(r.Content, "passwordKey"),
			// captchaKey:  mapDelAndGet(r.Content, "captchaKey"),
			Image:    image,
			ImageURL: iu,
			Ext:      r.Content,
		}, nil
	}
	return nil, nil
}

// Login 执行登录流程
func (f *Fetch) Login(key string, li *LoginInfo) (bool, error) {
	config, ok := f.Config[key]
	if !ok || config == nil {
		return false, errors.New("config not found")
	}
	if li.Username == "" {
		return false, errors.New("username is empty")
	}
	if li.Password == "" {
		return false, errors.New("password is empty")
	}
	if li.ImageURL != "" && li.Captcha == "" {
		return false, errors.New("captcha is empty")
	}
	var URL string
	if config.Login.PostURL != "" {
		URL = config.Login.PostURL
	} else if config.Login.URL != "" {
		URL = config.Login.URL
	}
	if URL == "" {
		return false, errors.New("url is empty")
	}

	data := make(url.Values)
	for k, v := range li.Ext {
		if strings.HasSuffix(k, "Key") {
			continue
		}
		switch k {
		case "username":
			v = li.Username
		case "password":
			v = li.Password
		case "captcha":
			v = li.Captcha
		}
		ct, ok := config.Login.Convert[k]
		if ok && ct != nil {
			v = convertString(v, ct)
		}
		kn, ok := li.Ext[k+"Key"]
		if ok {
			k = kn
		}
		data[k] = []string{v}
	}

	req, err := http.NewRequest("POST", config.Base+URL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.Header.Add("Referer", config.Base+config.Login.URL)
	for k, vs := range config.Login.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	cs := f.Cookie[key]
	for _, c := range cs {
		req.AddCookie(c)
	}

	// requestDump, err := httputil.DumpRequest(req, true)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(requestDump))
	// resp, err := http.DefaultClient.Do(nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	cs = updateCookies(resp.Cookies(), cs)

	contentType := resp.Header.Get("Content-Type")
	r, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return false, err
	}
	// fmt.Printf("%s", body)
	if config.Login.CheckLogin != "" {
		sb := string(body)
		if strings.Contains(sb, config.Login.CheckLogin) {
			return true, nil
		}
		return false, errors.New(sb)
	}
	return true, nil
}

// Index 获取入口数据
func (f *Fetch) Index(key string) (*Res, error) {
	v := f.Config[key]
	if v != nil {
		return f.Data(v.Base + v.Index.URL)
	}
	return nil, nil
}

// Data 获取指定URL数据
func (f *Fetch) Data(ref string) (*Res, error) {
	cr := matchConfigRule(ref, f.Config)
	if cr != nil {
		cs := f.Cookie[cr.Config.Key]
		// client := &http.Client{}
		req, err := http.NewRequest("GET", ref, nil)
		// req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.89 Safari/537.36")
		// req.Header.Add("Cookie", "cdb_onlineusernum=2658; cdb_sid=Ka7Guj;")
		for _, c := range cs {
			req.AddCookie(c)
		}
		resp, err := http.DefaultClient.Do(req)
		// resp, err := http.Get(ref)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		f.Cookie[cr.Config.Key] = updateCookies(resp.Cookies(), cs)

		contentType := resp.Header.Get("Content-Type")
		r, err := charset.NewReader(resp.Body, contentType)
		if err != nil {
			return nil, err
		}
		doc, err := goquery.NewDocumentFromReader(r)
		// doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			return nil, err
		}

		base, err := url.Parse(ref)
		if err != nil {
			return nil, err
		}

		var res *Res
		switch (*cr.Rule)["type"] {
		case "form":
			res = parseForm(base, cr, doc)
		case "index":
			res = parseIndex(base, cr.Rule, doc)
		case "list":
			res = parseList(base, cr.Rule, doc)
		case "thread":
			res = parseThread(base, cr.Rule, doc)
		}

		if cr.IsIndex && cr.Config.Index.Category != nil {
			rule := cr.Config.Index.Category
			doc.Find(rule.Items).Each(func(i int, s *goquery.Selection) {
				title := s.Text()
				res.Categories = append(res.Categories, map[string]string{
					"title": title,
					"link":  getLink(base, s, "href"),
				})
			})
		}
		return res, nil
	}
	return nil, nil
}

func matchConfigRule(url string, config map[string]*Config) *ConfigRule {
	for _, v := range config {
		if strings.HasPrefix(url, v.Base) {
			url = url[len(v.Base):]
			for _, r := range v.Rules {
				match := (*r)["match"]
				matched, err := regexp.MatchString("^"+match+"$", url)
				if err != nil {
					log.Println(err)
				} else if matched {
					return &ConfigRule{v, r, url == v.Index.URL}
				}
			}
		}
	}
	return nil
}

func parseForm(base *url.URL, cr *ConfigRule, doc *goquery.Document) *Res {
	res := &Res{Content: make(map[string]string)}
	rule := *cr.Rule
	for k, v := range rule {
		if k == "type" || k == "match" {
			continue
		}
		elem := doc.Find(v)
		switch goquery.NodeName(elem) {
		case "input":
			// inputName, _ := elem.Attr("name")
			// res.Content[k+"Key"] = inputName
			// inputValue, _ := elem.Attr("value")
			// res.Content[k] = inputValue
			res.Content[k+"Key"] = elem.AttrOr("name", "")
			res.Content[k] = elem.AttrOr("value", "")
		case "select":
			res.Content[k+"Key"] = elem.AttrOr("name", "")
			// res.Content[k] = elem.Find("option[checked]").AttrOr("value", "")
			s := elem.Find("option[checked]")
			sv := ""
			if len(s.Nodes) == 0 {
				// default
				sv = elem.Find("option").AttrOr("value", "") // first option
			} else {
				sv = s.AttrOr("value", "")
			}
			res.Content[k] = sv
		case "img":
			img, _ := elem.Attr("src")
			if img != "" {
				link, err := url.Parse(img)
				if err == nil {
					res.Content[k] = base.ResolveReference(link).String()
				}
			}
		default:
			elHTML, err := goquery.OuterHtml(elem)
			if err == nil {
				res.Content[k] = elHTML
			}
		}
	}
	return res
}

func parseIndex(base *url.URL, ruleMap *map[string]string, doc *goquery.Document) *Res {
	res := &Res{}
	rule := *ruleMap
	cat := rule["categories"]
	catTitle := rule["categoryTitle"]
	doc.Find(cat).Each(func(i int, s *goquery.Selection) {
		// key := fmt.Sprintf("key%v", i)
		key := "key" + strconv.Itoa(i)
		title := s.Find(catTitle)
		res.Categories = append(res.Categories, map[string]string{
			"key":   key,
			"title": title.Text(),
			"link":  getLink(base, title, "href"),
		})
		items := rule["items"]
		itemTitle := rule["itemTitle"]
		itemDesc := rule["itemDesc"]
		itemThreadTodayCount := rule["itemThreadTodayCount"]
		itemLastThread := rule["itemLastThread"]
		itemLastReply := rule["itemLastReply"]
		s.Find(items).Each(func(i int, s *goquery.Selection) {
			elem := s.Find(itemTitle)
			lastThread := s.Find(itemLastThread)
			lastReply := s.Find(itemLastReply)
			res.Items = append(res.Items, map[string]string{
				"categoryKey":      key,
				"title":            elem.Text(),
				"link":             getLink(base, elem, "href"),
				"desc":             s.Find(itemDesc).Text(),
				"threadTodayCount": s.Find(itemThreadTodayCount).Text(),
				"lastThread":       lastThread.Text(),
				"lastThreadLink":   getLink(base, lastThread, "href"),
				"lastReply":        lastReply.Text(),
				"lastReplyLink":    getLink(base, lastReply, "href"),
			})
		})
	})
	return res
}

func parseList(base *url.URL, ruleMap *map[string]string, doc *goquery.Document) *Res {
	res := &Res{}
	rule := *ruleMap
	items := rule["items"]
	itemTitle := rule["itemTitle"]
	itemAuthor := rule["itemAuthor"]
	itemAvatar := rule["itemAvatar"]
	itemLastReply := rule["itemLastReply"]
	itemReplyCount := rule["itemReplyCount"]
	doc.Find(items).Each(func(i int, s *goquery.Selection) {
		title := s.Find(itemTitle)
		author := s.Find(itemAuthor)
		lastReply := s.Find(itemLastReply)
		res.Items = append(res.Items, map[string]string{
			"title":         title.Text(),
			"link":          getLink(base, title, "href"),
			"author":        author.Text(),
			"authorLink":    getLink(base, author, "href"),
			"avatar":        getLink(base, s.Find(itemAvatar), "src"),
			"lastReply":     lastReply.Text(),
			"lastReplyLink": getLink(base, lastReply, "href"),
			"replyCount":    s.Find(itemReplyCount).Text(),
		})
	})
	return res
}

func parseThread(base *url.URL, ruleMap *map[string]string, doc *goquery.Document) *Res {
	res := &Res{Content: make(map[string]string)}
	rule := *ruleMap
	title := rule["title"]
	body := rule["body"]
	author := rule["author"]
	avatar := rule["avatar"]
	res.Content["title"] = doc.Find(title).Text()
	html, _ := doc.Find(body).Html()
	res.Content["body"] = html
	res.Content["author"] = doc.Find(author).Text()
	res.Content["avatar"] = getLink(base, doc.Find(avatar), "src")

	items := rule["items"]
	itemContent := rule["itemContent"]
	itemAuthor := rule["itemAuthor"]
	itemAvatar := rule["itemAvatar"]
	itemNo := rule["itemNo"]
	doc.Find(items).Each(func(i int, s *goquery.Selection) {
		content, _ := s.Find(itemContent).Html()
		author := s.Find(itemAuthor)
		res.Items = append(res.Items, map[string]string{
			"content":    content,
			"author":     author.Text(),
			"authorLink": getLink(base, author, "href"),
			"avatar":     getLink(base, s.Find(itemAvatar), "src"),
			"no":         s.Find(itemNo).Text(),
		})
	})
	return res
}

func getLink(base *url.URL, elem *goquery.Selection, attr string) string {
	ref, exists := elem.Attr(attr)
	if exists {
		link, err := url.Parse(ref)
		if err == nil {
			return base.ResolveReference(link).String()
		}
	}
	return ""
}

func updateCookies(from []*http.Cookie, to []*http.Cookie) []*http.Cookie {
	for _, cookie := range from {
		ix := -1
		for i, oc := range to {
			if oc.Name == cookie.Name {
				ix = i
				break
			}
		}
		if ix < 0 {
			to = append(to, cookie)
		} else {
			to[ix] = cookie
		}
	}
	return to
}

func convertString(s string, cts [][]string) string {
	for _, ct := range cts {
		if len(ct) < 1 {
			continue
		}
		switch ct[0] {
		case "hex":
			s = hex.EncodeToString([]byte(s))
		case "md5":
			// hasher := md5.New()
			// hasher.Write([]byte(s))
			// s = string(hasher.Sum(nil))
			hash := md5.Sum([]byte(s))
			s = hex.EncodeToString(hash[:])
		}
	}
	return s
}
