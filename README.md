# TitleFinder

TitleFinder is a tool written in Go that extracts webpage titles from a list of URLs. It supports loading URL lists from a file and can make requests through a proxy. Results can be output to a file or directly printed to the console.

## Features

- Reads URL list from a file
- Supports making requests through a proxy
- Extracts titles from each webpage
- Can output results to a file
- Defaults to printing results to the console
- Includes help command to display usage instructions

## Installation and Usage

### Build

To build the executable file, run the following command in the root directory of the project:

```bash
git clone https://github.com/xiaotsai/TitleFinder.git
go get github.com/PuerkitoBio/goquery
go build -o TitleFinder
```

```
Usage: TitleFinder.exe -l <file> [-o <output>] [-p <proxy>] [-t <threads>]

Options:
  -l <file>    Path to the input file containing URLs (required)
  -o <output>  Path to the output file (optional). If not provided, output will be printed to the console.
  -p <proxy>   Proxy URL to use for HTTP requests (optional). Format: [http://]host:port
               If protocol is not specified, http:// will be used by default.
  -t <threads> Number of concurrent threads (optional, default 10)
  -h           Display this help message
  ```
