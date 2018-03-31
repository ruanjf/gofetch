package gofetch

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {
	f, err := New("./rule/v2ex.yaml")
	if err != nil {
		t.Error(err)
		return
	}
	config, ok := f.Config["v2ex"]
	if !ok {
		t.Error("config not found")
		return
	}
	if config.Login.CheckLogin != "id=\"money\"" {
		t.Error("config not equals", config)
	}
}

func TestNewConfigPathNotFound(t *testing.T) {
	f, err := New("./rule/xxxx.yaml")
	if err != nil {
		t.Error(err)
	}
	_, ok := f.Config["xxxx"]
	if ok {
		t.Error("Config found")
	}
}

func TestNewConfigError(t *testing.T) {
	f, err := New("./testdata/error.yaml")
	if err != nil {
		t.Error(err)
	}
	_, ok := f.Config["error"]
	if ok {
		t.Error("Config found")
	}
}

func TestIndexNoData(t *testing.T) {
	f, err := New()
	if err != nil {
		t.Error(err)
	}
	res, err := f.Index("nodata")
	if err != nil {
		t.Error(err)
	}
	if res != nil {
		t.Error("have data")
	}
}

func TestIndexHipda(t *testing.T) {
	config, res, err := makeData(&testConfig{
		"./rule/hipda.yaml",
		"./testdata/hipda/index.html",
		"https://www.hi-pda.com/forum/index.php",
	}, func(f *Fetch, config *Config, relRef string) (*Res, error) {
		config.Index.URL = relRef
		return f.Index("hipda")
	})
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}
	l := len(res.Items)
	if l != 16 {
		t.Error("items len not equals 16:", l)
		return
	}
	item := res.Items[0]
	one := map[string]string{
		"categoryKey":      "key0",
		"title":            "Hi!PDA站务与公告",
		"link":             config.Base + "/forumdisplay.php?fid=5",
		"desc":             "Hi!PDA的站务，版面划分，申诉，建议等等。",
		"threadTodayCount": "1",
		"lastThread":       "我实名手机注册了，可是我想换手 ...",
		"lastThreadLink":   config.Base + "/redirect.php?tid=2229418&goto=lastpost#lastpost",
		"lastReply":        "孙月星",
		"lastReplyLink":    config.Base + "/space.php?username=%CB%EF%D4%C2%D0%C7",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("item not equals:", item)
		return
	}
	l = len(res.Categories)
	if l != 4 {
		t.Error("categories len not equals 4:", l)
		return
	}
	item = res.Categories[0]
	one = map[string]string{
		"key":   "key0",
		"title": "Hi! PDA",
		"link":  config.Base + "/index.php?gid=35",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("category not equals:", item)
		return
	}
}

func TestIndex(t *testing.T) {
	_, res, err := makeData(&testConfig{
		"./rule/v2ex.yaml",
		"./testdata/v2ex/tech.html",
		"https://www.v2ex.com/?tab=tech",
	}, func(f *Fetch, config *Config, relRef string) (*Res, error) {
		config.Index.URL = relRef
		return f.Index("v2ex")
	})
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}
	l := len(res.Items)
	if l != 50 {
		t.Error("len not equals 50:", l)
	}
	l = len(res.Categories)
	if l != 11 {
		t.Error("categories len not equals 11:", l)
		return
	}
	if res.Categories[0]["title"] != "技术" ||
		res.Categories[1]["title"] != "创意" ||
		res.Categories[2]["title"] != "好玩" {
		t.Error("categories not equals:", res.Categories)
		return
	}
}

func TestData(t *testing.T) {
	config, res, err := makeData(&testConfig{
		"./rule/v2ex.yaml",
		"./testdata/v2ex/tech.html",
		"https://www.v2ex.com/?tab=tech",
	}, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}
	l := len(res.Items)
	if l != 50 {
		t.Error("len not equals 50:", l)
		return
	}
	item := res.Items[0]
	one := map[string]string{
		"title":         "年会被耍了 感觉很没意思 所以接下来该干啥呢",
		"link":          config.Base + "/t/416297#reply214",
		"author":        "MrFireAwayH",
		"authorLink":    config.Base + "/member/MrFireAwayH",
		"avatar":        "http://v2ex.assets.uxengine.net/gravatar/e83ea61fcfd8fcd0493058406ae67fa9?s=48&d=retro",
		"lastReply":     "MrFireAwayH",
		"lastReplyLink": config.Base + "/member/MrFireAwayH",
		"replyCount":    "214",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("item not equals:", item)
		return
	}
}

func TestDataHipda(t *testing.T) {
	config, res, err := makeData(&testConfig{
		"./rule/hipda.yaml",
		"./testdata/hipda/forumdisplay.html",
		"https://www.hi-pda.com/forum/forumdisplay.php?fid=5",
	}, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}
	l := len(res.Items)
	if l != 65 {
		t.Error("len not equals 65:", l)
	}
	item := res.Items[1]
	one := map[string]string{
		"title":         "希望尽快支持境外手机号码的验证",
		"link":          config.Base + "/viewthread.php?tid=2238815&extra=page%3D1",
		"author":        "wencan",
		"authorLink":    config.Base + "/space.php?uid=723094",
		"avatar":        "",
		"lastReply":     "五家渠",
		"lastReplyLink": config.Base + "/space.php?username=%CE%E5%BC%D2%C7%FE",
		"replyCount":    "3",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("item not equals:", item)
	}
}

func TestDataThread(t *testing.T) {
	config, res, err := makeData(&testConfig{
		"./rule/v2ex.yaml",
		"./testdata/v2ex/thread.html",
		"https://www.v2ex.com/t/416297#reply214",
	}, nil)

	if err != nil {
		t.Error(err)
		return
	}

	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}

	l := len(res.Items)
	if l != 13 {
		t.Error("len not equals 13:", l)
		return
	}

	if res.Content["title"] != "苹果应用新规？哪位 IOS 开发者解读一下。" ||
		res.Content["author"] != "imswing" ||
		res.Content["avatar"] != "http://v2ex.assets.uxengine.net/avatar/dbd7/e290/80834_large.png?m=1468303764" {
		t.Error("content not equals:", res.Content)
		return
	}

	item := res.Items[0]
	one := map[string]string{
		"content":    "如果法规不允许才要提供",
		"author":     "SingingZhou",
		"authorLink": config.Base + "/member/SingingZhou",
		"avatar":     "http://v2ex.assets.uxengine.net/avatar/48ce/aeb8/77219_normal.png?m=1413354603",
		"no":         "1",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("item not equals:", item)
		return
	}
}

func TestDataHipdaThread(t *testing.T) {
	config, res, err := makeData(&testConfig{
		"./rule/hipda.yaml",
		"./testdata/hipda/viewthread.html",
		"https://www.hi-pda.com/forum/viewthread.php?tid=2229418&extra=page%3D1",
	}, nil)

	if err != nil {
		t.Error(err)
		return
	}

	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}

	l := len(res.Items)
	if l != 4 {
		t.Error("len not equals 4:", l)
		return
	}

	if res.Content["title"] != "" ||
		res.Content["author"] != "" ||
		res.Content["avatar"] != "" {
		t.Error("content not equals:", res.Content)
		return
	}

	item := res.Items[0]
	one := map[string]string{
		"content":    "因为一些无法明言的原因，希望论坛能支持境外手机号码的验证<br/>\n谢谢 ",
		"author":     "wencan",
		"authorLink": config.Base + "/space.php?uid=723094",
		"avatar":     "https://www.hi-pda.com/forum/uc_server/data/avatar/000/72/30/94_avatar_middle.jpg",
		"no":         "1",
	}
	if !reflect.DeepEqual(one, item) {
		t.Error("item 0 not equals:", item)
		return
	}
}

func TestCreateLoginInfo(t *testing.T) {
	_, _, err := makeData(&testConfig{
		"./rule/v2ex.yaml",
		"./testdata/v2ex/login.html",
		"https://www.v2ex.com/signin",
	}, func(f *Fetch, config *Config, relRef string) (*Res, error) {
		res, err := f.CreateLoginInfo("v2ex")
		if err != nil {
			t.Error(err)
			return nil, nil
		}

		if res == nil {
			t.Error("empty", res)
			return nil, nil
		}

		if res.ImageURL != config.Base+"/_captcha?once=71137" ||
			res.Ext["usernameKey"] != "f54101a3479d5c787e99735b5b7f6f7f0cd03985fc7452e12bd770d7e12b2afe" ||
			res.Ext["passwordKey"] != "af8e36ec2f8c3848984fb011c417dd0a9304bdd972d8d76c0ca89f4473b18069" ||
			res.Ext["captchaKey"] != "49ccd07e395876f0abb65acbe042e6ec6a94b5f32c760ff2090a55d602f7168e" ||
			res.Ext["once"] != "71137" {
			t.Error("content not equals:", res.Ext)
			return nil, nil
		}
		return nil, nil
	})

	if err != nil {
		t.Error(err)
		return
	}
}

func TestCreateLoginInfoHipda(t *testing.T) {
	_, _, err := makeData(&testConfig{
		"./rule/hipda.yaml",
		"./testdata/hipda/login.html",
		"https://www.hi-pda.com/forum/logging.php?action=login",
	}, func(f *Fetch, config *Config, relRef string) (*Res, error) {
		res, err := f.CreateLoginInfo("hipda")
		if err != nil {
			t.Error(err)
			return nil, nil
		}

		if res == nil {
			t.Error("empty", res)
			return nil, nil
		}

		if res.Ext["usernameKey"] != "username" ||
			res.Ext["passwordKey"] != "password" ||
			res.Ext["sid"] != "ez3xC5" ||
			res.Ext["formhash"] != "1f111148" ||
			res.Ext["referer"] != "" ||
			res.Ext["loginfield"] != "username" ||
			res.Ext["questionid"] != "0" ||
			res.Ext["answer"] != "" ||
			res.Ext["cookietime"] != "2592000" {
			t.Error("content not equals:", res.Ext)
			return nil, nil
		}
		return nil, nil
	})

	if err != nil {
		t.Error(err)
		return
	}
}

func xxxTestLoginV2ex(t *testing.T) {
	key := "v2ex"
	f, err := New("./rule/" + key + ".yaml")
	if err != nil {
		t.Error(err)
		return
	}
	li, err := f.CreateLoginInfo(key)
	if err != nil {
		t.Error(err)
		return
	}
	li.Username = "abc"
	li.Password = "def"
	li.Captcha = "ghi"
	ok, err := f.Login(key, li)
	if err != nil {
		t.Error(err)
		return
	}
	if ok {
		t.Error("login must be failed")
		return
	}
}

func xxxTestLoginHipda(t *testing.T) {
	key := "hipda"
	f, err := New("./rule/" + key + ".yaml")
	if err != nil {
		t.Error(err)
		return
	}
	li, err := f.CreateLoginInfo(key)
	if err != nil {
		t.Error(err)
		return
	}
	li.Username = "ruanjf"
	li.Password = "JUUBEAcP7J8cZbDG"
	ok, err := f.Login(key, li)
	if err != nil {
		t.Error(li, err)
		return
	}
	if !ok {
		t.Error(li, "login failed")
		return
	}

	res, err := f.Index(key)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil || len(res.Items) == 0 {
		t.Error("empty", res)
		return
	}
	l := len(res.Items)
	if l != 16 {
		t.Error("items len not equals 16:", l)
		return
	}
}

type testConfig struct {
	config,
	data,
	url string
}

func makeData(tc *testConfig, fn func(*Fetch, *Config, string) (*Res, error)) (*Config, *Res, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Add("Content-Type", "text/html; charset=utf-8")
		w.Header().Add("Set-Cookie", "cdb_onlineusernum=2658; expires=Tue, 26-Dec-2017 15:01:38 GMT; Max-Age=300; path=/")
		w.Header().Add("Set-Cookie", "cdb_sid=Ka7Guj; expires=Tue, 02-Jan-2018 14:56:38 GMT; Max-Age=604800; path=/; httponly")
		file, err := os.Open(tc.data) // For read access.
		if err != nil {
			println("read file error:", err)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	}))
	defer ts.Close()

	f, _ := New(tc.config)

	cr := matchConfigRule(tc.url, f.Config)
	if cr == nil {
		return nil, nil, errors.New("not found Config Rule")
	}
	config := cr.Config
	// ref, _ := url.Parse(tc.url)
	// relRef := ref.RequestURI()
	relRef := tc.url[len(config.Base):]
	(*cr.Rule)["match"] = regexp.QuoteMeta(relRef)
	config.Base = ts.URL

	var res *Res
	var err error
	if fn != nil {
		res, err = fn(f, config, relRef)
	} else {
		res, err = f.Data(config.Base + relRef)
	}
	return config, res, err
}
