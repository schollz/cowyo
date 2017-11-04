
<p align="center">
<img
    src="/static/img/logo.png"
    width="260" height="80" border="0" alt="linkcrawler">
<br>
<a href="https://travis-ci.org/schollz/cowyo"><img src="https://img.shields.io/travis/schollz/cowyo.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/schollz/cowyo/releases/latest"><img src="https://img.shields.io/badge/version-2.8.0-brightgreen.svg?style=flat-square" alt="Version"></a>
</p>

<p align="center">A feature-rich wiki for minimalists</a></p>

*cowyo* is a self-contained wiki server that makes jotting notes easy and _fast_. The most important feature here is _simplicity_. Other features include versioning, page locking, self-destructing messages, encryption, and listifying. You can [download *cowyo* as a single executable](https://github.com/schollz/cowyo/releases/latest) or install it with Go. Try it out at https://cowyo.com.

There is now [a command-line tool, *cowyodel*](https://github.com/schollz/cowyodel) to interact with *cowyo* and transfer information between computers with only a code phrase: [schollz/cowyodel](https://github.com/schollz/cowyodel).

Getting Started
===============

## Install

If you have go

```
go get -u github.com/schollz/cowyo/...
```

or just [download the latest release](https://github.com/schollz/cowyo/releases/latest).

## Run

To run just double click or from the command line:

```
cowyo
```

and it will start a server listening  on `0.0.0.0:8050`. To view it, just go to http://localhost:8050 (the server prints out the local IP for your info if you want to do LAN networking). You can change the port with `-port X`, and you can listen *only* on localhost using `-host localhost`.

### Running with TLS

Specify a matching pair of SSL Certificate and Key to run cowyo using https. *cowyo* will now run in a secure session. 

*N.B. Let's Encrypt is a CA that signs free and signed certificates.*

```
cowyo --cert "/path/to/server.crt" --key "/p/t/server.key"
```

### Running with Docker

You can easily get started with Docker. First pull the latest image and create the volume with:

```
docker run -d --name cowyo -p 8050:8050 schollz/cowyo
```

Then you can stop it with `docker stop cowyo` and start it again with `docker start cowyo`.

### Running as a micro-CMS

There are a couple of command-line flags that you can use to make your own micro-CMS. 

```
cowyo -lock 123 -default-page index.html -css mystyle.css
```

The `-lock` flag will automatically lock every page with the passphrase "123". Also, the default behavior will be to redirect `/` to `/index.html`. Also, every page that is published will automatically redirect to `/mypage/read` which will show the custom CSS file if it is supplied with `-css`.

## Usage

*cowyo* is straightforward to use. Here are some of the basic features:

### Editing

When you open a document you'll be directed to an alliterative animal (which is supposed to be easy to remember). You can write in Markdown. Saving is performed as soon as you stop writing. You can easily link pages using [[PageName]] as you edit.

![Editing](http://i.imgur.com/vEs2U8z.gif)

### History

You can easily see previous versions of your documents.

![History](http://i.imgur.com/CxhRkyo.gif)

### Lists

You can easily make lists and check them off.

![Lists](http://i.imgur.com/7xbauy8.gif)

### Locking

Locking prevents other users from editing your pages without a passphrase.

![Locking](http://i.imgur.com/xwUFV8b.gif)

### Encryption

Encryption is performed using AES-256.

![Encryption](http://i.imgur.com/rWoqoLB.gif)

### Self-destructing pages

Just like in mission impossible.

![Self-destructing](http://i.imgur.com/upMxFQh.gif)

## Development

You can run the tests using

```
$ cd $GOPATH/src/github.com/schollz/cowyo
$ go test ./...
```

Any contributions are welcome.

## License

MIT

