# imgo

'imgo' is a command for scrapeing images.

## Caution
Be sure to check each page to see if scraping is allowed. If scraping is not allowed, do not use this command. Also, please refrain from scraping in a way that will overload the server. In this command, the interval between HTTP requests is spaced out in order to reduce the load. By default, it is set to 3 seconds, so set it to an appropriate value.

We are not responsible for any damage caused by the use of this command. Thank you for your understanding.

## Installing
```sh
$ go get -u github.com/EscherWF/imgo
```

## Usage

For details of each command, please refer to the `--help` .

```
Usage:
  imgo [flags]
  imgo [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  help        Help about any command
  version

Flags:
  -c, --cookies stringArray   You can set multiple cookies. For example, -c key1:value1 -c key2:value2 ...
  -d, --delay int             Specify the number of seconds between image requests. (default 3)
      --dest string           Specify the directory to output the images. (default "./")
  -h, --help                  help for imgo
  -l, --limit int             Specify the maximum number of images to save. (default 256)
      --parallel int          Specify the number of parallel HTTP requests. (default 5)
  -u, --user string           Specify the information for BASIC authentication. For example, username:password.
  -v, --verbose               verbose

```

## TODO
- Golang Test.
- Support for dynamically rendered pages.
- Support for retrieving images from stylesheets.
- And more.

## Dependency
- [cobra](https://github.com/spf13/cobra)
- [colly](https://github.com/gocolly/colly)
