package cmd

import (
	"encoding/base64"
	"log"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/spf13/cobra"
)

type RootFlag struct {
	cookies  []string
	dest     string
	delay    int
	limit    int
	parallel int
	user     string
	verbose  bool
}

var rootFlag RootFlag

var rootCmd = &cobra.Command{
	Use:   "imgo",
	Short: "short description.",
	Long:  "Long description.\nLong description.",
	Args:  cobra.MinimumNArgs(1),
	Run:   mainRun,
}

var userAgent = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	"AppleWebKit/537.36 (KHTML, like Gecko)",
	"Chrome/91.0.4472.124 Safari/537.36",
}

func mainRun(cmd *cobra.Command, args []string) {

	var seq = 0
	var url = args[0]

	if rootFlag.delay < 0 || rootFlag.limit < 0 || rootFlag.parallel < 0 {
		panic("The options should be positive integers.")
	}

	// Check the existence of the directory
	_, err := os.Stat(rootFlag.dest)
	if os.IsNotExist(err) {
		panic(err.Error())
	}

	// Basic Autenticate
	header := http.Header{}
	if rootFlag.user != "" {
		user := strings.TrimSpace(rootFlag.user)
		pattAuth := regexp.MustCompile(`^\S+[^\s:]+:[^\s:]+\S+$`)
		if !pattAuth.MatchString(user) {
			panic("The format of the given auth info is not correct.")
		}

		auth := base64.StdEncoding.EncodeToString([]byte(user))
		header.Set("Authorization", "Basic "+auth)
	}

	// Cookies
	var cookieList []*http.Cookie
	if len(rootFlag.cookies) > 0 {
		for _, cookie := range rootFlag.cookies {
			kv := strings.Split(strings.TrimSpace(cookie), ":")
			if len(kv) != 2 {
				panic("The format of the given cookie is not correct.")
			}

			cookieList = append(cookieList, &http.Cookie{Name: kv[0], Value: kv[1]})
		}
	}

	c := colly.NewCollector(
		colly.UserAgent(strings.Join(userAgent, " ")),
		colly.Async(true),
	)

	c.OnHTML("img, source", func(e *colly.HTMLElement) {
		for _, url := range *imageUrls(e) {
			c.Visit(e.Request.AbsoluteURL(url.url))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println(r.Headers)
		pattBs64 := regexp.MustCompile(`image\/([\S\D]+);base64,`)
		if pattBs64.MatchString(r.URL.Opaque) && rootFlag.limit > seq {
			r.Abort()

			var extention = "unknown"
			contentType := "image/" + pattBs64.FindStringSubmatch(r.URL.Opaque)[1]
			exts, _ := mime.ExtensionsByType(contentType)
			if len(exts) > 0 {
				extention = exts[0]
			}

			bs64 := pattBs64.ReplaceAllString(r.URL.Opaque, "")
			dec, err := base64.StdEncoding.DecodeString(bs64)
			if err != nil {
				log.Println(err.Error())
			}

			seq++

			f, err := os.Create("base64Image_" + strconv.Itoa(seq) + "." + extention)
			if err != nil {
				log.Println(err.Error())
			}
			defer f.Close()

			_, err = f.Write(dec)
			if err != nil {
				log.Println(err.Error())
			}
			err = f.Sync()
			if err != nil {
				log.Println(err.Error())
			}
		}
	})

	c.OnResponse(func(r *colly.Response) {
		contentType := r.Headers.Get("content-type")
		if !strings.Contains(contentType, "image/") {
			return
		}

		if rootFlag.limit > seq {
			li := newlocalImage(r.FileName(), contentType)
			err := r.Save(rootFlag.dest + li.basename)
			if err != nil {
				log.Println(err.Error())
			}

			seq++

			if rootFlag.verbose {
				log.Println("response url", r.Request.URL, r.StatusCode)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Error:", err, r.Request.URL)
	})

	// Delay & Parallel connections
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: rootFlag.parallel,
		Delay:       time.Duration(rootFlag.delay) * time.Second,
	})
	c.SetCookies(url, cookieList)
	c.Request("GET", url, nil, nil, header)
	c.Wait()
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

type imageUrl struct {
	url string
}

func (iu *imageUrl) __init() {
	pattUrl := regexp.MustCompile(` [0-9]*[wx]?$`)
	iu.url = pattUrl.ReplaceAllString(iu.url, "")
	iu.url = strings.TrimSpace(iu.url)
}

func newImageUrl(url string) *imageUrl {
	iu := &imageUrl{url: url}
	iu.__init()
	return iu
}

type localImage struct {
	basename    string
	contentType string
}

func (iu *localImage) __init() {
	var extention = ""
	exts, _ := mime.ExtensionsByType(iu.contentType)
	iu.basename = colly.SanitizeFileName(iu.basename)

	// TODO: ここ直す
	switch len(exts) {
	case 1:
		extention = exts[0]
	case 2:
		extention = exts[1]
	case 3:
		extention = exts[1]
	}

	pattUrl := regexp.MustCompile(`.unknown$`)
	iu.basename = pattUrl.ReplaceAllString(iu.basename, extention)
	pattUrl = regexp.MustCompile(`.*` + extention + `$`)

	if pattUrl.MatchString(iu.basename) {
		return
	}

	iu.basename += extention

}

func newlocalImage(basename string, contentType string) *localImage {
	li := &localImage{
		basename:    basename,
		contentType: contentType,
	}
	li.__init()
	return li
}

func imageUrls(e *colly.HTMLElement) *[]imageUrl {
	var urls []imageUrl
	src := e.Attr("src")
	urls = append(urls, *newImageUrl(src))

	srcset := e.Attr("srcset")
	srcsets := strings.Split(srcset, ",")

	for _, url := range srcsets {
		urls = append(urls, *newImageUrl(url))
	}

	return &urls
}

func init() {
	cobra.OnInitialize(initConfig)

	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.StringArrayVarP(&rootFlag.cookies, "cookies", "c", []string{}, "You can set multiple cookies. For example, -c key1:value1 -c key2:value2 ...")
	persistentFlags.StringVar(&rootFlag.dest, "dest", "./", "Specify the directory to output the images.")
	persistentFlags.IntVarP(&rootFlag.delay, "delay", "d", 0, "Specify the number of seconds between image requests.")
	persistentFlags.IntVarP(&rootFlag.limit, "limit", "l", 256, "Specify the maximum number of images to save.")
	persistentFlags.IntVar(&rootFlag.parallel, "parallel", 5, "Specify the number of parallel HTTP requests.")
	persistentFlags.StringVarP(&rootFlag.user, "user", "u", "", "Specify the information for BASIC authentication. For example, username:password.")
	persistentFlags.BoolVarP(&rootFlag.verbose, "verbose", "v", false, "verbose")
}

func initConfig() {
	// if cfgFile != "" {
	// 	// Use config file from the flag.
	// 	viper.SetConfigFile(cfgFile)
	// } else {
	// 	// Find home directory.
	// 	home, err := os.UserHomeDir()
	// 	cobra.CheckErr(err)

	// 	// Search config in home directory with name ".cobra" (without extension).
	// 	viper.AddConfigPath(home)
	// 	viper.SetConfigType("yaml")
	// 	viper.SetConfigName(".cobra")
	// }

	// viper.AutomaticEnv()

	// if err := viper.ReadInConfig(); err == nil {
	// 	fmt.Println("Using config file:", viper.ConfigFileUsed())
	// }
}
