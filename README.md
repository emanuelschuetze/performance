# OpenSlides Performance

This is a small script written in Go to test performance of OpenSlides.


## Install and run

Install [Go (>=1.6)](https://golang.org/) and setup your GOPATH environment
variable (e. g. to `$HOME/Go`).

Open command line and run

    go get github.com/OpenSlides/performance
    cd $GOPATH/src/github.com/OpenSlides/performance
    go run main.go

To get help on command line options run

    go run main.go --help


## Development

To manage dependencies of this project we use the
[submodule](https://git-scm.com/docs/git-submodule) feature of Git
featuring [Vendetta](https://github.com/dpw/vendetta).

    go get github.com/dpw/vendetta
    $GOPATH/bin/vendetta


## License and authors

OpenSlides Performance is Free/Libre Open Source Software (FLOSS), and
distributed under the MIT License, see `LICENSE` file. The authors of
OpenSlides Performance are mentioned in the `AUTHORS` file.
