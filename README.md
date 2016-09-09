![Logo](/static/img/cowyo.png)

# [cowyo.com](http://cowyo.com/)

[![Version 1.1.0](https://img.shields.io/badge/version-1.1.0-brightgreen.svg)]() [![Go Report Card](https://goreportcard.com/badge/github.com/schollz/cowyo)](https://goreportcard.com/report/github.com/schollz/cowyo) [![Join the chat at https://gitter.im/schollz/cowyo](https://badges.gitter.im/schollz/cowyo.svg)](https://gitter.im/schollz/cowyo?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This is a self-contained notepad webserver that makes sharing easy and _fast_. The most important feature here is _simplicity_. There are many other features as well including versioning, page locking, self-destructing messages, encryption, math support, syntax highlighting, command line support, content-delivery, and listifying. Read on to learn more about the features.

# Features

**Simplicity**. The philosophy here is to _just type_. To jot a note, simply load the page at [`/`](http://cowyo.com/) and just start typing. No need to press edit, the browser will already be focused on the text. No need to press save - it will automatically save when you stop writing. The URL at [`/`](http://cowyo.com/) will redirect to an easy-to-remember name that you can use to reload the page at anytime, anywhere. But, you can also use any URL you want, e.g. [`/AnythingYouWant`](http://cowyo.com/AnythingYouWant). All pages can be rendered into HTML by adding `/view`. For example, the page [`/AnythingYouWant`](http://cowyo.com/AnythingYouWant) is rendered at [`/AnythingYouWant/view`](http://cowyo.com/AnythingYouWant/view). You can write in HTML or [Markdown](https://daringfireball.net/projects/markdown/) for page rendering. To quickly link to `/view` pages, just use `[[AnythingYouWant]]`.

![Simply type to edit.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help1.gif)

<br>

**Listifying**. If you are writing a list and you want to tick off things really easily, just add `/list`. For example, after editing [`/grocery`](http://cowyo.com/grocery), goto [`/grocery/list`](http://cowyo.com/grocery/list). In this page, whatever you click on will be struck through and moved to the end. This is helpful if you write a grocery list and then want to easily delete things from it.

![Lists are easy to make.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help2.gif)

<br>

**Page locking**. Pages can be locked by providing a password to prevent further editing. The whole version tree will still be available. _Note_: This is not available for list mode.

![Locking is easy.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help3.gif)

<br>

**Automatic versioning**. All previous versions of all notes are stored and can be accessed by adding `?version=X` onto `/view` or `/edit`. If you are on the `/view` or `/edit` pages the menu below will show the most substantial changes in the history. Note, only the _current_ version can be edited (no branching allowed, yet).

![Versioning is easy.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help4.gif)

<br>

**Self-destructing messages**. You can write a message that will delete itself when a user loads it (in any view). Useful for transmitting sensitive information. To use, simply add a line somewhere that says only "`self-destruct`".

![Mission impossible style self-destruction.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help5.gif)

<br>

**Security**. HTTPS support is provided and everything is sanitized to prevent XSS attacks. Though all URLs are publicly accessible, you are free to obfuscate your website by using an obscure/random address (read: the site is still publicly accessible, just hard to find!). In addition to TLS support, you can PGP-encrypt your messages using a passphrase (_Note: This will delete the version tree_).

![Security and encryption baked in.](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help6.gif)

<br>

**Syntax highlighting**. If you use a coding extension (e.g. .py, .md, .txt, .js, ...) then you'll automatically see syntax highlighting and line numbers.

![Coding syntax is provided if you use an extension](https://raw.githubusercontent.com/schollz/cowyo/master/static/img/help7.gif)

<br>

**CLI tools**. Want to upload/download from the command line? Its super easy. Upload/download files using `curl` with a simple command:

```bash
$ echo "Hello, world!" > hi.txt
$ curl -L --upload-file hi.txt cowyo.com
  File uploaded to http://cowyo.com/hi.txt
$ curl -L cowyo.com/test.txt
  Hello, world!
```

or just skip the file-creation step and let `cowyo` figure out a name for you:

```bash
$ echo "Wow, so easy" | curl -L --upload-file "-" cowyo.com
  File uploaded to http://cowyo.com/CautiousCommonLoon
$ curl -L cowyo.com/CautiousCommonLoon
  Wow, so easy
```

<br>

**Content Delivery**. Want use a script on your own domain? Just use the extension `/raw` with optional versioning (e.g. `/raw?version=1`). `cowyo` will serve these files with a wildcard Access-Control-Allow-Origin so they will work anywhere. Check out a live example [on this JSFiddle](https://jsfiddle.net/9mm3afao/).

**Keyboard Shortcuts**. Quickly transition between Edit/View/List by using `Ctl+Shift+E` to Edit, `Ctl+Shift+Z` to View, and `Ctl+Shift+L` to Listify.

**Admin controls**. The Admin can view/delete all the documents by setting the `-a YourAdminKey` when starting the program. Then the admin has access to the `/ls/YourAdminKey` to view and delete any of the pages.

**Math support**. Math is supported with [Katex](https://github.com/Khan/KaTeX) using `$\frac{1}{2}$` for inline equations and `$$\frac{1}{2}$$` for regular equations.

# Install

## From release

Just [download the latest release](https://github.com/schollz/cowyo/releases/tag/1.1.0), unzip and run.

## From source

First [install Go 1.6+](https://golang.org/doc/install).

```
$ git clone https://github.com/schollz/cowyo.git
$ cd cowyo
$ make
$ ./cowyo
```

## From Docker

```
$ docker pull schollz/cowyo
$ docker run -it -p 8003:8003 -v /local/dir/to/store/data:/data schollz/cowyo
```

# Contact

If you'd like help or you find a bug, please submit [an issue](https://github.com/schollz/cowyo/issues). Any other comments, questions or anything at all, just [tweet me @zack_118](https://twitter.com/intent/tweet?screen_name=zack_118)

# Acknowledgements

Thanks to [tscholl2](https://github.com/tscholl2) and [sjsafranek](https://github.com/sjsafranek).

Icons made by [Freepik](http://www.freepik.com) from [www.flaticon.com](http://www.flaticon.com), licensed by [CC 3.0 BY](http://creativecommons.org/licenses/by/3.0/ "Creative Commons BY 3.0").

File uploading from [transfer.sh](https://github.com/dutchcoders/transfer.sh/blob/98399c91dd86682077cf9542badbf1658fd9a8c1/transfersh-server/handlers.go#L293-L369), licensed by [MIT license](https://github.com/dutchcoders/transfer.sh/blob/40c9bf7675fb84e78d9a011052b9d0900ec7dde1/LICENSE).

# License

The MIT License (MIT)

Copyright (c) 2016 Zack

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
