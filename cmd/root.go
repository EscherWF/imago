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

	"github.com/gocolly/colly"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

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
	ext, _ := mime.ExtensionsByType(iu.contentType)
	iu.basename = colly.SanitizeFileName(iu.basename)

	if len(ext) > 0 {
		pattUrl := regexp.MustCompile(`.unknown$`)
		iu.basename = pattUrl.ReplaceAllString(iu.basename, ext[0])
	}
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
	var seq = 0

	userAgent := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"AppleWebKit/537.36 (KHTML, like Gecko)",
		"Chrome/91.0.4472.124 Safari/537.36",
	}

	c := colly.NewCollector(
		colly.UserAgent(strings.Join(userAgent, " ")),
	)

	hdr := http.Header{}
	hdr.Set("Authorization", "Basic "+basicAuth("test", "test"))

	c.OnHTML("img, source", func(e *colly.HTMLElement) {
		for _, url := range *imageUrls(e) {
			c.Visit(e.Request.AbsoluteURL(url.url))
		}
	})

	c.OnHTML("*[style]", func(e *colly.HTMLElement) {
		style := e.Attr("style")
		pattUrl := regexp.MustCompile(`.*background-image:url\((.*)\).*`)
		styleImage := pattUrl.FindStringSubmatch(style)
		if styleImage == nil {
			return
		}

		iu := newImageUrl(styleImage[0])
		c.Visit(e.Request.AbsoluteURL(iu.url))
	})

	c.OnRequest(func(r *colly.Request) {
		pattBs64 := regexp.MustCompile(`image\/([\S\D]+);base64,`)
		if isBs64 := pattBs64.MatchString(r.URL.Opaque); isBs64 {
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

			seq += 1
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
		if isImage := strings.Contains(contentType, "image/"); !isImage {
			return
		}

		li := newlocalImage(r.FileName(), contentType)
		err := r.Save("./" + li.basename)
		if err != nil {
			log.Println(err.Error())
		}
		// for debugging
		log.Println("response url", r.Request.URL, r.StatusCode)
	})

	// c.OnError(func(r *colly.Response, err error) {
	// 	log.Println("Request URL:", r.Request.URL, "\nError:", err)
	// 	// unautorizedなら、認証オプションの使用を勧める
	// })

	// リクエスト間隔（Delay）
	// c.Limit(&colly.LimitRule{
	// 	DomainGlob: "*",
	// 	Delay:      10 * time.Second,
	// })

	c.Request("GET", args[0], nil, nil, hdr)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().StringP("author", "a", "YOUR NAME", "author name for copyright attribution")
	rootCmd.PersistentFlags().StringVarP(&userLicense, "license", "l", "", "name of license for the project")
	rootCmd.PersistentFlags().Bool("viper", true, "use Viper for configuration")
	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	viper.SetDefault("license", "apache")

	// rootCmd.AddCommand(addCmd)
	// rootCmd.AddCommand(initCmd)
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
