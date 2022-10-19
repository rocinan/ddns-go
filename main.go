package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var (
	name          = flag.String("n", "", "name")
	domain        = flag.String("d", "", "domain name")
	apiKey        = flag.String("k", "", "api key")
	apiSecret     = flag.String("s", "", "api secret")
	checkInterval = flag.Int("i", 60, "check interval in seconds")

	GDHOST = func(name, domain string) string {
		return fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/A/%s", domain, name)
	}
	GDBODY = func(name, ip string) string {
		return fmt.Sprintf(`
		[
			{
			  "data": "%s",
			  "name": "%s",
			  "port": 1,
			  "priority": 0,
			  "protocol": "string",
			  "service": "string",
			  "ttl": 600,
			  "type": "A",
			  "weight": 0
			}
		]`, ip, name)
	}
)

func main() {
	flag.Parse()
	if *name == "" || *domain == "" || *apiKey == "" || *apiSecret == "" {
		flag.Usage()
		return
	}
	g := &GDDDNS{
		Host:      GDHOST(*name, *domain),
		APIKey:    *apiKey,
		APISecret: *apiSecret,
	}
	ticker := time.NewTicker(time.Duration(*checkInterval) * time.Second)
	for range ticker.C {
		if ip, err := GetIp(); err != nil {
			log.Println("获取外部ip失败: ", err)
			continue
		} else {
			if rip, err := g.Query(); err != nil {
				log.Println("获取dns记录ip失败: ", err)
				continue
			} else {
				if ip == rip {
					log.Println("未检测到ip变化等待下次检测")
				} else {
					log.Println("检测到ip变化,开始更新记录: ", rip, " -> ", ip)
					if err := g.Update(ip); err != nil {
						log.Println("更新dns记录失败: ", err)
						continue
					} else {
						log.Println("更新dns记录成功")
					}
				}
			}
		}
	}
}

type GDDDNS struct {
	Host      string
	APIKey    string
	APISecret string
}

func (g *GDDDNS) Query() (string, error) {
	req, _ := http.NewRequest("GET", g.Host, nil)
	req.Header.Add("Authorization", fmt.Sprintf("sso-key %s:%s", g.APIKey, g.APISecret))
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return "", err
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("status code: %d", resp.StatusCode)
		}
		if res, err := io.ReadAll(resp.Body); err != nil {
			return "", fmt.Errorf("read body error: %s", err)
		} else {
			reso := gjson.ParseBytes(res).Array()
			if len(reso) == 0 {
				return "", nil
			}
			return reso[0].Get("data").String(), nil
		}
	}
}

func (g *GDDDNS) Update(ip string) error {
	req, _ := http.NewRequest("PUT", g.Host, strings.NewReader(GDBODY(*name, ip)))
	req.Header.Add("Authorization", fmt.Sprintf("sso-key %s:%s", g.APIKey, g.APISecret))
	req.Header.Add("Content-Type", "application/json")
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return err
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code: %d", resp.StatusCode)
		}
		return nil
	}
}

func GetIp() (string, error) {
	req, _ := http.NewRequest("GET", "https://api.ip.sb/ip", nil)
	req.Header.Add("User-Agent", "Mozilla")
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return "", err
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("status code: %d", resp.StatusCode)
		}
		if res, err := io.ReadAll(resp.Body); err != nil {
			return "", fmt.Errorf("read body error: %s", err)
		} else {
			return strings.TrimSpace(string(res)), nil
		}
	}
}
