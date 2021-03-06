# ddclassify

ddclassify - Dewey Decimal Classifier

## Synopsis

Usage:
    go run ddclassify.go [-h|--help]
                         [-t title [-a author]]
                         [-i isbn]
                         [-d path [-p pattern] [-r] [-e directories] [-c /path/to/library [-m 1|2|4|8|17|18|20|24]] [-misfile]]
                         [-x /path/to/ddc.xml]
                         [-depth 1..3]
                         [-g]
                         [-v]

## Examples

Search by title and author:

    $ go run ddclassify.go -t "Alice's Adventures in Wonderland" -a "Lewis Carroll"

Search by title and author using the Google Books API to lookup the ISBN:

    $ go run ddclassify.go -t "Alice's Adventures in Wonderland" -a "Lewis Carroll" -g

Search by ISBN:

    $ go run ddclassify.go -i 9780060081393

Search filename (Title - Author.ext):

    $ go run ddclassify.go -d "/path/to/Alice's Adventures in Wonderland - Lewis Carroll.epub"

Search filename (Title - Author.ext) and move to library:

    $ go run ddclassify.go -d "/path/to/Alice's Adventures in Wonderland - Lewis Carroll.epub" -c /path/to/library -m 8

Specify regular expression pattern for parsing title and author from filenames.

    $ go run ddclassify.go -d ... -p "^(?P<title>.+?)(?:(_|,).*)? - (?P<author>.+)\.([A-Za-z]+)$"

Search directory recursively and exclude search directories:

    $ go run ddclassify.go -d /path/to/library -r -e music,movies

Create empty DDC directory structure /tmp/eBooks without transferring files:

    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks

Create DDC directory structure /tmp/eBooks and transfer files:

    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 1
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 2
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 4
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 8

Create DDC directory structure /tmp/eBooks transfer and rename files:

    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 17 # 1 + 16
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 18 # 2 + 16
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 20 # 4 + 16
    $ go run ddclassify.go -d /path/to/library -r -c /tmp/eBooks -m 24 # 8 + 16

Find misfiled entries in search directory:

    $ go run ddclassify.go -d /path/to/library -r -misfile

Specify DDC structure XML file mapping Dewey Decimal numbers to descriptions:

    $ go run ddclassify.go ... -x /path/to/ddc.xml

Specify classification depth, [1..3], indicating class, division, or section.

    $ go run ddclassify.go ... -depth 1

## See Also

https://en.wikipedia.org/wiki/List_of_Dewey_Decimal_classes

https://www.oclc.org/developer/develop/web-services/classify/classification.en.html

https://developers.google.com/books/docs/v1/using
