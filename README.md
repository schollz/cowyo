![Logo](/static/img/cowyo.png)

# CowYo - [Demo](http://cowyo.com/)

[![Version 0.94](https://img.shields.io/badge/version-0.94-brightgreen.svg)]() [![Go Report Card](https://goreportcard.com/badge/github.com/schollz/cowyo)](https://goreportcard.com/report/github.com/schollz/cowyo) [![Join the chat at https://gitter.im/schollz/cowyo](https://badges.gitter.im/schollz/cowyo.svg)](https://gitter.im/schollz/cowyo?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This is a self-contained notepad webserver that makes sharing easy and _fast_. The most important feature here is *simplicity*. There are many other features as well including versioning, page locking, self-destructing messages, encryption, math support, and listifying. Read on to learn more about the features.

# Features
**Simplicity**. The philosophy here is to *just type*. To jot a note, simply load the page at [`/`](http://cowyo.com/) and just start typing. No need to press edit, the browser will already be focused on the text. No need to press save - it will automatically save when you stop writing. The URL at [`/`](http://cowyo.com/) will redirect to an easy-to-remember name that you can use to reload the page at anytime, anywhere. But, you can also use any URL you want, e.g. [`/AnythingYouWant`](http://cowyo.com/AnythingYouWant). All pages can be rendered into HTML by adding `/view`. For example, the page [`/AnythingYouWant`](http://cowyo.com/AnythingYouWant) is rendered at [`/AnythingYouWant/view`](http://cowyo.com/AnythingYouWant/view). You can write in HTML or [Markdown](https://daringfireball.net/projects/markdown/) for page rendering. To quickly link to `/view` pages, just use `[[AnythingYouWant]]`.

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


**Keyboard Shortcuts**. Quickly transition between Edit/View/List by using `Ctl+Shift+E` to Edit, `Ctl+Shift+Z` to View, and `Ctl+Shift+L` to Listify.

**Admin controls**. The Admin can view/delete all the documents by setting the `-a YourAdminKey` when starting the program. Then the admin has access to the `/ls/YourAdminKey` to view and delete any of the pages.

**Math support**. Math is supported with [Katex](https://github.com/Khan/KaTeX) using `$\frac{1}{2}$` for inline equations and `$$\frac{1}{2}$$` for regular equations.



# Install

First [install Go](https://golang.org/doc/install). Then continue.

## To access locally...

Then, if you want to host on your local network just do:

```
git clone https://github.com/schollz/cowyo.git
cd cowyo
make
./cowyo -p :8001 LOCALIPADDRESS
```

and then goto the address `http://LOCALIPADDRESS:8001/`

## To access from anywhere...

For this you need to forward port 80 and [get a DNS for your external address](https://www.duckdns.org/). I recommend using `NGINX` as middleware, as it will do caching of the static files for you.

```bash
sudo apt-get install nginx
```

There is an example `NGINX` block in `install/`. If you want to use SSL instead, follow the instructions in `letsencrypt/README.md`. To automatically install, on Raspberry Pi / Ubuntu / Debian system use:

```
git clone https://github.com/schollz/cowyo.git
cd cowyo
nano Makefile <--- EDIT this Makefile to include YOUR EXTERNAL ADDRESS
make && sudo make install
```

Now the program starts and stops with

```
sudo /etc/init.d/cowyo start|stop|restart
```

Edit your crontab (`sudo crontab -e`) to start on boot:

```
@reboot /etc/init.d/cowyo start
```

# Usage

```
$ cowyo --help
cowyo: A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'cowyo localhost'
Example: 'cowyo -p :8080 localhost:8080'
Example: 'cowyo -db /var/lib/cowyo/db.bolt localhost:8003'
Example: 'cowyo -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
Options:
  -a string
        key to access admin priveleges (default no admin priveleges)
  -crt string
        location of ssl crt
  -db string
        location of database file (default "/home/mu/cowyo/data.db")
  -httptest.serve string
        if non-empty, httptest.NewServer serves on this address and blocks
  -key string
        location of ssl key
  -p string
        port to bind (default ":8003")
```

If you set the admin flag, `-a` you can access a list of all the current files by going to `/ls/WhateverYouSetTheFlagTo`.

# Contact
If you'd like help, go ahead and clone and send a pull request. If you find a bug, please submit [an issue](https://github.com/schollz/cowyo/issues). Any other comments, questions or anything at all, just <a href="https://twitter.com/intent/tweet?screen_name=zack_118" class="twitter-mention-button" data-related="zack_118">tweet me @zack_118</a>

# Contributors
Thanks to [tscholl2](https://github.com/tscholl2).
