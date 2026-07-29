# About cowyo

cowyo is a minimalist shared scratchpad. Every page lives at a simple URL: open one, type, and your text is saved automatically. When other people are editing the same page, their carets appear as dim gray lines.

The cow menu provides the page's controls:

- Copy the complete page text to the clipboard.
- Lock or unlock editing.
- Encrypt or decrypt text with a password.
- Publish or unpublish the page.
- Switch between light and dark themes.
- Make the page self-destruct after its next load.

## Visibility, locks, and encryption

These features serve different purposes:

Unpublished is the default. It keeps a page out of cowyo's sitemap and asks search engines not to index it, but it does not prevent someone with the URL from opening it. Publishing makes the page discoverable to search engines; unpublishing does not change its text.

Page locking prevents changes until the page-lock password is entered. It does not hide or encrypt the text. A locked page remains readable, and its password cannot be recovered if forgotten. The password is sent to cowyo when you lock or unlock the page, but is never stored as plain text.

Encryption protects the text itself. Encryption and decryption happen in your browser, and the encryption password is never sent to a server. The server stores the protected content as an encrypted block. Ordinary text before or after an encrypted block is preserved when that block is decrypted. If you lose the encryption password, the encrypted text cannot be recovered.

Unlock a page before encrypting it or changing whether it is published.

## Self-destructing pages

The bomb action arms a page to self-destruct. The next browser or curl GET receives the page one final time and deletes it; later visits open an empty page. A HEAD request does not consume it.

Arming self-destruct automatically unpublishes the page. It cannot be combined with publishing, and you must unlock a locked page before arming or cancelling it.

## Using cowyo with curl

A normal curl GET returns only the stored text, without the browser
editor:

```sh
curl https://cowyo.com/my-notes
```

POST to the home page to create a new randomly named page:

```sh
curl --data-binary @notes.txt https://cowyo.com/
```

POST to a named path to create that page or replace its complete
contents:

```sh
curl --data-binary @notes.txt https://cowyo.com/my-notes
```

For a short note, the text can be supplied directly:

```sh
curl --data-binary 'remember the milk' https://cowyo.com/shopping-list
```

`--data-binary` preserves the text exactly, including line breaks.
