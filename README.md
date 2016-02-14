![Logo](https://i.imgur.com/ixnBYOl.png)

# AwwKoala - [Demo](http://awwkoala.com/)
## A Websocket Wiki and Kind Of A List Application
[![Version 1.0](https://img.shields.io/badge/version-1.0-brightgreen.svg)]() [![Go Report Card](https://goreportcard.com/badge/github.com/schollz/AwwKoala)](https://goreportcard.com/report/github.com/schollz/AwwKoala) [![Join the chat at https://gitter.im/schollz/AwwKoala](https://badges.gitter.im/schollz/AwwKoala.svg)](https://gitter.im/schollz/AwwKoala?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This is a self-contained wiki webserver that makes sharing easy and _fast_. You can make any page you want, and any page is editable by anyone. Pages load instantly for editing, and have special rendering for whether you want to view as a web page or view as list.

# Features
## Simplicity
The philosophy here is to *just type*. To jot a note, simply load the page at [`/`](http://AwwKoala.com/) and just start typing. No need to press edit, the browser will already be focused on the text. No need to press save - it will automatically save when you stop writing. The URL at [`/`](http://AwwKoala.com/) will redirect to an easy-to-remember name that you can use to reload the page at anytime, anywhere. But, you can also use any URL you want, e.g. [`/AnythingYouWant`](http://AwwKoala.com/AnythingYouWant).

## Viewing
All pages can be rendered into HTML by adding `/view`. For example, the page [`/AnythingYouWant`](http://AwwKoala.com/AnythingYouWant) is rendered at [`/AnythingYouWant/view`](http://AwwKoala.com/AnythingYouWant/view). You can write in HTML or [Markdown](https://daringfireball.net/projects/markdown/) for page rendering. To quickly link to `/view` pages, just use `[[AnythingYouWnat]]`. Math is supported with [Katex](https://github.com/Khan/KaTeX) using `$\frac{1}{2}$` for inline equations and `$$\frac{1}{2}$$` for regular equations.

## Listifying
If you are writing a list and you want to tick off things really easily, just add `/list`. For example, after editing [`/grocery`](http://AwwKoala.com/grocery), goto [`/grocery/list`](http://AwwKoala.com/grocery/list). In this page, whatever you click on will be striked through and moved to the end. This is helpful if you write a grocery list and then want to easily delete things from it.

## Automatic versioning
All previous versions of all notes are stored and can be accessed by adding `?version=X` onto `/view` or `/edit`. If you are on the `/view` or `/edit` pages the menu below will show the most substantial changes in the history. Note, only the _current_ version can be edited (no branching allowed, yet).

## Security

Now comes with HTTPS!

## Keyboard Shortcuts

Quickly transition between Edit/View/List by using `Ctl+Shift+E` to Edit, `Ctl+Shift+Z` to View, and `Ctl+Shift+L` to Listify.

## Admin controls

The Admin can view/delete all the documents by setting the `-a YourAdminKey` when starting the program. Then the admin has access to the `/ls/YourAdminKey` to view and delete any of the pages.

# Install
To get started on your local network just do:

```
git clone https://github.com/schollz/awwkoala.git
cd awwkoala
make
./awwkoala -p :8001 LOCALIPADDRESS
```

and then goto the address `http://LOCALIPADDRESS:8001/`

## Production server
I recommend using `NGINX` as middleware, as it will do caching of the static files for you. There is an example `NGINX` block in `install/`. To automatically install, on Raspberry Pi / Ubuntu / Debian system use:

```
git clone https://github.com/schollz/awwkoala.git
cd awwkoala
nano Makefile <--- EDIT this Makefile to include YOUR EXTERNAL ADDRESS
make && sudo make install
```

Now the program starts and stops with

```
sudo /etc/init.d/AwwKoala start|stop|restart
```

Edit your crontab (`sudo crontab -e`) to start on boot:

```
@reboot /etc/init.d/AwwKoala start
```

# Usage

```
$ awwkoala --help
awwkoala: A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'awwkoala localhost'
Example: 'awwkoala -p :8080 localhost:8080'
Example: 'awwkoala -db /var/lib/awwkoala/db.bolt localhost:8003'
Example: 'awwkoala -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
Options:
  -a string
        key to access admin priveleges (default no admin priveleges)
  -crt string
        location of ssl crt
  -db string
        location of database file (default "/home/mu/awwkoala/data.db")
  -httptest.serve string
        if non-empty, httptest.NewServer serves on this address and blocks
  -key string
        location of ssl key
  -p string
        port to bind (default ":8003")
```

If you set the admin flag, `-a` you can access a list of all the current files by going to `/ls/WhateverYouSetTheFlagTo`.

# Contact
If you'd like help, go ahead and clone and send a pull request. If you find a bug, please submit [an issue](https://github.com/schollz/AwwKoala/issues). Any other comments, questions or anything at all, just <a href="https://twitter.com/intent/tweet?screen_name=zack_118" class="twitter-mention-button" data-related="zack_118">tweet me @zack_118</a>

# Contributors
Thanks to [tscholl2](https://github.com/tscholl2).
