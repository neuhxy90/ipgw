package logout

import (
	"fmt"
	"ipgw/base/cfg"
	"ipgw/base/ctx"
	"ipgw/share"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func logoutWithSID(x *ctx.Ctx) (ok bool) {
	if x.Net.SID == "" {
		return false
	}

	if cfg.FullView {
		fmt.Printf(usingSID, x.Net.SID)
	}
	resp, err := share.Kick(x.Net.SID)
	share.ErrWhenReqHandler(err)
	body := share.ReadBody(resp)

	if cfg.FullView {
		fmt.Println(body)
	}

	if body == "下线请求发送失败" {
		if cfg.FullView {
			fmt.Fprintf(os.Stderr, failLogoutBySID, x.Net.SID)
		}
		return false
	}

	fmt.Println(successLogoutBySID)
	return true
}

func logoutWithUP(x *ctx.Ctx) {
	client := ctx.GetClient()

	if cfg.FullView {
		fmt.Printf(usingUP, x.User.Username)
	}

	// 请求获得必要参数
	resp, err := client.Get("https://pass.neu.edu.cn/tpass/login?service=https%3A%2F%2Fipgw.neu.edu.cn%2Fsrun_cas.php%3Fac_id%3D1")
	if err != nil {
		if cfg.FullView {
			fmt.Fprintf(os.Stderr, errWhenReadLT, err)
		}
		fmt.Fprintln(os.Stderr, errNetwork)
		os.Exit(2)
	}

	// 读取响应内容
	body := share.ReadBody(resp)

	// 读取lt post_url
	ltExp := regexp.MustCompile(`name="lt" value="(.+?)"`)
	lt := ltExp.FindAllStringSubmatch(body, -1)[0][1]

	if cfg.FullView {
		fmt.Printf(successGetLT, lt)
	}

	// 拼接data
	data := "rsa=" + x.User.Username + x.User.Password + lt +
		"&ul=" + strconv.Itoa(len(x.User.Username)) +
		"&pl=" + strconv.Itoa(len(x.User.Password)) +
		"&lt=" + lt +
		"&execution=e1s1" +
		"&_eventId=submit"

	// 构造请求
	req, _ := http.NewRequest("POST", "https://pass.neu.edu.cn/tpass/login?service=https%3A%2F%2Fipgw.neu.edu.cn%2Fsrun_cas.php%3Fac_id%3D1", strings.NewReader(data))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "pass.neu.edu.cn")
	req.Header.Add("Origin", "https://pass.neu.edu.cn")
	req.Header.Add("Referer", "https://pass.neu.edu.cn/tpass/login?service=https%3A%2F%2Fipgw.neu.edu.cn%2Fsrun_cas.php%3Fac_id%3D3")
	if x.UA != "" {
		req.Header.Add("User-Agent", x.UA)
	}

	// 发送请求
	resp, err = client.Do(req)

	share.ErrWhenReqHandler(err)

	// 读取响应内容
	body = share.ReadBody(resp)

	// 检查标题
	t := share.GetTitle(body)
	if t == "智慧东大--统一身份认证" {
		fmt.Fprintln(os.Stderr, wrongUOrP)
		os.Exit(2)
	}

	var id, sid string
	if strings.Contains(body, "aaa") {
		id, sid = share.GetIDAndSIDWhenCollision(body)
		if id == "" {
			fmt.Fprintln(os.Stderr, wrongState)
			os.Exit(2)
		}

		if sid == "" {
			fmt.Fprintln(os.Stderr, failGetInfo)
			os.Exit(2)
		}

		if cfg.FullView {
			fmt.Printf(successGetID, id)
		}

		if cfg.FullView {
			fmt.Printf(beginLogout, id)
		}

		// 踢下线
		resp, err := share.Kick(sid)

		share.ErrWhenReqHandler(err)
		body = share.ReadBody(resp)

		if cfg.FullView {
			fmt.Println(body)
		}

		if body != "下线请求已发送" {
			fmt.Fprintf(os.Stderr, failLogout, id)
			os.Exit(2)
		}

		resp, err = client.Get("https://ipgw.neu.edu.cn/srun_cas.php?ac_id=1")

		share.ErrWhenReqHandler(err)

		// 读取响应内容
		body = share.ReadBody(resp)

		out := share.GetIfOut(body)
		if out {
			fmt.Printf(successLogout, id)
			os.Exit(0)
		}

		share.GetIPAndSID(body, x)
	} else {
		out := share.GetIfOut(body)
		if out {
			fmt.Println(balanceOut)
			os.Exit(0)
		}

		// 读取IP与SID
		ok := share.GetIPAndSID(body, x)
		if !ok {
			fmt.Fprintln(os.Stderr, failGetInfo)
			os.Exit(2)
		}
	}

	resp, err = share.Kick(x.Net.SID)

	share.ErrWhenReqHandler(err)

	body = share.ReadBody(resp)

	if body != "下线请求已发送" {
		fmt.Fprintf(os.Stderr, failLogout, id)
		os.Exit(2)
	}

	if id == "" {
		fmt.Printf(successLogout, x.User.Username)
	} else {
		fmt.Printf(successLogout, id)
	}
}

func logoutWithC(x *ctx.Ctx) (ok bool) {
	client := ctx.GetClient()

	if cfg.FullView {
		fmt.Printf(usingC, x.User.Cookie.Value)
	}

	// 请求获得必要参数
	client.Jar.SetCookies(&url.URL{
		Scheme: "https",
		Host:   "ipgw.neu.edu.cn",
	}, []*http.Cookie{x.User.Cookie})

	resp, err := client.Get("https://ipgw.neu.edu.cn/srun_cas.php?ac_id=1")

	share.ErrWhenReqHandler(err)

	// 读取响应内容
	body := share.ReadBody(resp)

	// 检查标题
	t := share.GetTitle(body)
	if t == "智慧东大--统一身份认证" {
		if cfg.FullView {
			fmt.Fprintln(os.Stderr, failCookieExpired)
		}
		return false
	}

	// 不同账号登陆
	var id, sid string
	if strings.Contains(body, "aaa") {
		id, sid = share.GetIDAndSIDWhenCollision(body)
		if id == "" {
			fmt.Fprintln(os.Stderr, wrongState)
			os.Exit(2)
		}

		if sid == "" {
			fmt.Fprintln(os.Stderr, failGetInfo)
			os.Exit(2)
		}

		if cfg.FullView {
			fmt.Printf(successGetID, id)
		}

		if cfg.FullView {
			fmt.Printf(beginLogout, id)
		}

		// 踢下线
		resp, err := share.Kick(sid)

		share.ErrWhenReqHandler(err)
		body = share.ReadBody(resp)

		if cfg.FullView {
			fmt.Println(body)
		}

		if body != "下线请求已发送" {
			fmt.Fprintf(os.Stderr, failLogout, id)
			os.Exit(2)
		}

		resp, err = client.Get("https://ipgw.neu.edu.cn/srun_cas.php?ac_id=1")

		share.ErrWhenReqHandler(err)

		// 读取响应内容
		body = share.ReadBody(resp)

		share.GetIPAndSID(body, x)
	} else {
		// 读取学号
		usernameExp := regexp.MustCompile(`user_name" style="float:right;color: #894324;">(.+?)</span>`)
		username := usernameExp.FindAllStringSubmatch(body, -1)

		if len(username) < 1 {
			fmt.Fprintln(os.Stderr, failGetInfo)
			os.Exit(2)
		}
		x.User.Username = username[0][1]
		if cfg.FullView {
			fmt.Printf(successGetID, x.User.Username)
		}

		share.GetIPAndSID(body, x)

		if cfg.FullView {
			fmt.Println(sendingRequest)
		}
	}

	resp, err = share.Kick(x.Net.SID)

	share.ErrWhenReqHandler(err)

	body = share.ReadBody(resp)

	if body != "下线请求已发送" {
		fmt.Fprintf(os.Stderr, failLogout, id)
		os.Exit(2)
	}

	if id == "" {
		fmt.Printf(successLogout, x.User.Username)
	} else {
		fmt.Printf(successLogout, id)
	}
	return true
}