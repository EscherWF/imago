package cmd

import (
	"encoding/base64"
	"fmt"
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
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "imgo",
		Short: "short description.",
		Long:  "Long description.\nLong description.",
		Args:  cobra.MinimumNArgs(1),
		Run:   mainRun,
	}
)

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

func basicAuth(user, pass string) string {
	auth := user + ":" + pass
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func mainRun(cmd *cobra.Command, args []string) {

	cookies, _ := cmd.Flags().GetStringArray("cookie")
	dest, _ := cmd.Flags().GetString("dest")
	delay, _ := cmd.Flags().GetInt("delay")
	limit, _ := cmd.Flags().GetInt("limit")
	parallel, _ := cmd.Flags().GetInt("parallel")
	user, _ := cmd.Flags().GetString("user")
	isVerbose, _ := cmd.Flags().GetBool("verbose")

	var seq = 0

	userAgent := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"AppleWebKit/537.36 (KHTML, like Gecko)",
		"Chrome/91.0.4472.124 Safari/537.36",
	}

	c := colly.NewCollector(
		colly.UserAgent(strings.Join(userAgent, " ")),
		colly.Async(true),
	)

	// Delay & Parallel connections
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallel,
		Delay:       time.Duration(delay) * time.Second,
	})

	// Cookies
	if len(cookies) > 0 {
		var cookieList []*http.Cookie
		for _, cookie := range cookies {
			info := strings.Split(cookie, ":")
			cookieList = append(cookieList,
				&http.Cookie{
					Name:  info[0],
					Value: info[1],
				})
		}
		c.SetCookies(args[0], cookieList)
	}

	hdr := http.Header{}
	// Basic Autenticate
	if user != "" {
		info := strings.Split(user, ":")
		hdr.Set("Authorization", "Basic "+basicAuth(info[0], info[1]))
	}

	c.OnHTML("img, source", func(e *colly.HTMLElement) {
		for _, url := range *imageUrls(e) {
			c.Visit(e.Request.AbsoluteURL(url.url))
		}
	})

	// TODO: background-imageからも画像取得したい
	// c.OnHTML("*[style]", func(e *colly.HTMLElement) {
	// 	style := e.Attr("style")
	// 	pattUrl := regexp.MustCompile(`.*background-image:url\((.*)\).*`)
	// 	styleImage := pattUrl.FindStringSubmatch(style)
	// 	if styleImage == nil {
	// 		return
	// 	}

	// 	iu := newImageUrl(styleImage[0])
	// 	c.Visit(e.Request.AbsoluteURL(iu.url))
	// })

	c.OnRequest(func(r *colly.Request) {
		pattBs64 := regexp.MustCompile(`image\/([\S\D]+);base64,`)
		if pattBs64.MatchString(r.URL.Opaque) && limit > seq {
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

		if limit > seq {
			li := newlocalImage(r.FileName(), contentType)
			// TODO: destのIOエラーハンドリング
			err := r.Save(dest + li.basename)
			if err != nil {
				log.Println(err.Error())
			}
			seq++
		}
		if isVerbose {
			log.Println("response url", r.Request.URL, r.StatusCode)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "\nError:", err)
	})

	c.Request("GET", args[0], nil, nil, hdr)
	c.Wait()
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringArrayP("cookie", "c", []string{}, "You can set multiple cookies. For example, -c key1:value1 -c key2:value2 ...")
	rootCmd.PersistentFlags().String("dest", "./", "Specify the directory to output the images.")
	rootCmd.PersistentFlags().IntP("delay", "d", 0, "Specify the number of seconds between image requests.")
	rootCmd.PersistentFlags().IntP("limit", "l", 256, "Specify the maximum number of images to save.")
	rootCmd.PersistentFlags().Int("parallel", 5, "Specify the number of parallel HTTP requests.")
	rootCmd.PersistentFlags().StringP("user", "u", "", "Specify the information for BASIC authentication. For example, username:password.")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cobra")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
