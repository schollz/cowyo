![Logo](https://i.imgur.com/ixnBYOl.png)

# AwwKoala
## A Websocket Wiki and Kind Of A List Application
![Version 1.0](https://img.shields.io/badge/version-1.0-brightgreen.svg)

This is a self-contained wiki webserver that makes sharing easy and _fast_. You can make any page you want, and any page is editable by anyone. Pages load instantly for editing, and have special rendering for whether you want to view as a web page or view as list.

# Features
## Just type.
To jot a note, simply load the page at [`/`](/) and just start typing. No need to press edit, the browser will already be focused on the text. No need to press save - it will automatically save when you stop writing. The URL at [`/`](/) will redirect to an easy-to-remember name that you can use to reload the page at anytime, anywhere. But, you can also use any URL you want, e.g. [`/AnythingYouWant`](/AnythingYouWant).

## Views
All pages can be rendered into HTML by adding `/view`. For example, the page [`/AnythingYouWant`](/AnythingYouWant) is rendered at [`/AnythingYouWant/view`](/AnythingYouWant/view). You can write in HTML or [Markdown](https://daringfireball.net/projects/markdown/) for page rendering. To quickly link to `/view` pages, just use `&#91;&#91;AnythingYouWant&#93;&#93;`. Math is supported with [Katex](https://github.com/Khan/KaTeX) using `&#36;\frac{1}{2}&#36;` for inline equations and `&#36;&#36;\frac{1}{2}&#36;&#36;` for regular equations.

## Lists
If you are writing a list and you want to tick off things really easily, just add `/list`. For example, after editing [`/grocery`](/grocery), goto [`/grocery/list`](/grocery/list). In this page, whatever you click on will be striked through and moved to the end. This is helpful if you write a grocery list and then want to easily delete things from it.

## Automatic versioning
All previous versions of all notes are stored and can be accessed by adding `?version=X` onto `/view` or `/edit`. If you are on the `/view` or `/edit` pages the menu below will show the most substantial changes in the history. Note, only the _current_ version can be edited (no branching allowed, yet).

## Keyboard Shortcuts

Quickly transition between Edit/View/List by using `Ctl+Shift+E` to Edit, `Ctl+Shift+Z` to View, and `Ctl+Shift+L` to Listify.

# Contact
Any other comments, questions or anything at all, just <a href="https://twitter.com/intent/tweet?screen_name=zack_118" class="twitter-mention-button" data-related="zack_118">tweet me @zack_118</a>

Have fun.

**Powered by Raspberry Pi, Go, and NGINX**

![Raspberry Pi](/static/img/raspberrypi.png) ![Go Mascot](/static/img/gomascot.png) ![Nginx](/static/img/nginx.png)
