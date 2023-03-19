package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
)

var (
	tpl_file string
	tpl      *template.Template
	role     string
	minutes  int
	myip     string
	hostname string

	dns_config, _ = dns.ClientConfigFromFile("/etc/resolv.conf")
	dns_client    = new(dns.Client)

	cat_zone = "he.okerr.com."
	cat_host = "cat.he.okerr.com."
)

func resolveA(host string, ns string) ([]string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, net.JoinHostPort(ns, dns_config.Port))
		},
	}
	ip, err := r.LookupHost(context.Background(), host)
	//ip, err := r.LookupNetIP(context.Background(), "udp4", host)
	return ip, err

}

func resolveNS(zone string) ([]string, error) {

	var nslist []string

	m := new(dns.Msg)
	m.SetQuestion(zone, dns.TypeNS)
	m.RecursionDesired = true
	r, _, err := dns_client.Exchange(m, net.JoinHostPort(dns_config.Servers[0], dns_config.Port))
	if err != nil {
		log.Println("DNS ERR", err)
		return nil, err
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, err
	}

	for _, a := range r.Answer {
		if host, ok := a.(*dns.NS); ok {
			parts := strings.Split(host.String(), "\t")
			srv := parts[len(parts)-1]
			nslist = append(nslist, srv)
		}
	}
	return nslist, nil
}

func prepare(c *gin.Context) map[string]string {
	//var buf bytes.Buffer
	dt := time.Now()
	_, now_mins, _ := dt.Clock()
	var status string
	var left int
	var alist []string

	if now_mins >= minutes {
		status = "ERR"
		left = 0
	} else {
		status = "OK"
		left = minutes - now_mins
	}

	nslist, err := resolveNS(cat_zone)
	check(err)

	for {
		idx := rand.Intn(len(nslist))

		nsiplist, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip4", nslist[idx])
		check(err)

		nsip := nsiplist[0].String()
		log.Println("Resolve", cat_host, "via ns #", idx, nsip)
		fmt.Printf("nslist: %v", nsiplist)

		alist, err = resolveA(cat_host, nsip)

		if err == nil {
			// resolved ok
			break
		} else {
			log.Println("ResolveA error:", err)
		}
	}

	check(err)

	timestr := dt.Format("01-02-2006 15:04:05")

	m := map[string]string{
		"role":    role,
		"host":    hostname,
		"myip":    myip,
		"timestr": timestr,
		"status":  status,
		"left":    strconv.Itoa(left),
		"nsname":  nslist[0],
		"catip":   alist[0]}

	return m

}

func json(c *gin.Context) {

	m := prepare(c)

	//tpl.Execute(&buf, ctx)
	//c.String(http.StatusOK, buf.String())
	c.IndentedJSON(http.StatusOK, m)
}

func index(c *gin.Context) {

	m := prepare(c)

	c.HTML(http.StatusOK, path.Base(tpl_file), m)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func parse_args() {
	def_role := getenv("ROLE", "main")
	def_template := getenv("TEMPLATE", "/etc/okerr/cat.html.tmpl")
	sys_hostname, _ := os.Hostname()
	def_hostname := getenv("HOSTNAME", sys_hostname)
	//def_minutes, _ := strconv.Atoi(getenv("MINUTES", "0"))

	flag.StringVar(&role, "r", def_role, "Role: main/backup/sorry")
	flag.StringVar(&tpl_file, "t", def_template, "Location of HTML template")
	flag.StringVar(&hostname, "n", def_hostname, "hostName")
	//flag.IntVar(&minutes, "m", def_minutes, "hostName")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"okerr cat (with gin!)\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if role == "main" {
		minutes = 20
	} else if role == "backup" {
		minutes = 40
	} else {
		// sorry server always OK
		minutes = 60
	}

	log.Println("Template:", tpl_file, "Role:", role)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	var err error

	parse_args()

	dns_client.Net = "udp4"

	tpl, err = template.ParseFiles(tpl_file)
	check(err)

	res, err := http.Get("https://ifconfig.me/")
	check(err)
	defer res.Body.Close()

	check(err)
	bs := make([]byte, 1014)
	n, err := res.Body.Read(bs)
	myip = string(bs[:n])

	fmt.Println("IP:", myip)

	r := gin.Default()
	r.LoadHTMLFiles(tpl_file)
	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.GET("/", index)
	r.GET("/json", json)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	rand.Seed(time.Now().Unix())
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
