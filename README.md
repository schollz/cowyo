
<p align="center">
<img
    src="/static/img/logo.png"
    width="260" height="80" border="0" alt="linkcrawler">
<br>
<a href="https://travis-ci.org/schollz/boltdb-server"><img src="https://img.shields.io/travis/schollz/boltdb-server.svg?style=flat-square" alt="Build Status"></a>
<img src="https://img.shields.io/badge/version-2.0.6-brightgreen.svg?style=flat-square" alt="Version">
</p>

<p align="center">A feature-rich wiki for minimalists</a></p>

*cowyo* is a self-contained wiki server that makes jotting notes easy and _fast_. The most important feature here is _simplicity_. Other features include versioning, page locking, self-destructing messages, encryption, and listifying. You can [download *cowyo* as a single executable](https://github.com/schollz/cowyo/releases/latest) or install it with Go.

Getting Started
===============

## Install

If you have go

```
go get github.com/schollz/cowyo
```

or just download from the [latest releases](https://github.com/schollz/cowyo/releases/latest).

## Run

To run just double click or from the command line:

```
cowyo
```

## Usage

*cowyo* is straightforward to use. Here are some of the basic features:

### Editing

When you open a document you'll be directed to an alliterative animal (which is supposed to be easy to remember). You can write in Markdown. Saving is performed as soon as you stop writing.

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

## License

MIT