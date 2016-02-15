![Logo](https://i.imgur.com/ixnBYOl.png)

# AwwKoala
## A Websocket Wiki and Kind Of A List Application
![Version 0.9](https://img.shields.io/badge/version-0.9-brightgreen.svg)

This is a self-contained wiki webserver that makes sharing easy and _fast_. You can make any page you want, and any page is editable by anyone. Pages load instantly for editing, and have special rendering for whether you want to view as a web page or view as list. **AwwKoala** is also [Open Source](https://github.com/schollz/AwwKoala).

## Features
**Simplicity**. The philosophy here is to *just type*. To jot a note, simply load the page at [`/`](/) and just start typing. No need to press edit, the browser will already be focused on the text. No need to press save - it will automatically save when you stop writing. The URL at [`/`](/) will redirect to an easy-to-remember name that you can use to reload the page at anytime, anywhere. But, you can also use any URL you want, e.g. [`/AnythingYouWant`](/AnythingYouWant).

**Viewing**. All pages can be rendered into HTML by adding `/view`. For example, the page [`/AnythingYouWant`](/AnythingYouWant) is rendered at [`/AnythingYouWant/view`](/AnythingYouWant/view). You can write in HTML or [Markdown](https://daringfireball.net/projects/markdown/) for page rendering. To quickly link to `/view` pages, just use `&#91;&#91;AnythingYouWant&#93;&#93;`. Math is supported with [Katex](https://github.com/Khan/KaTeX) using `&#36;\frac{1}{2}&#36;` for inline equations and `&#36;&#36;\frac{1}{2}&#36;&#36;` for regular equations.

**Listifying**. If you are writing a list and you want to tick off things really easily, just add `/list`. For example, after editing [`/grocery`](/grocery), goto [`/grocery/list`](/grocery/list). In this page, whatever you click on will be struck through and moved to the end. This is helpful if you write a grocery list and then want to easily delete things from it.

**Security**. HTTPS support is provided and everything is sanitized to prevent XSS attacks. Though all URLs are publicly accessible, you are free to obfuscate your website by using an obscure/random address (read: the site is still publicly accessible, just hard to find!). The automatic URL is an alliterative animal description - of which there are over 500,000 possibilities - so the URL is easy to remember and hard to guess.

**Keyboard Shortcuts**. Quickly transition between Edit/View/List by using `Ctl+Shift+E` to Edit, `Ctl+Shift+Z` to View, and `Ctl+Shift+L` to Listify.


# Contact
Any other comments, questions or anything at all, just <a href="https://twitter.com/intent/tweet?screen_name=zack_118" class="twitter-mention-button" data-related="zack_118">tweet me @zack_118</a>

Have fun.

**Powered by Raspberry Pi, Go, and NGINX**

![Raspberry Pi](/static/img/raspberrypi.png) ![Go Mascot](/static/img/gomascot.png) ![Nginx](/static/img/nginx.png)
