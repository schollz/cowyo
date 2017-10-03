# versionedtext

<a href="https://travis-ci.org/schollz/versionedtext"><img src="https://img.shields.io/travis/schollz/versionedtext.svg?style=flat-square" alt="Build Status"></a>
<img src="https://img.shields.io/badge/coverage-90%25-green.svg?style=flat-square" alt="Code Coverage">
<a href="https://godoc.org/github.com/schollz/versionedtext">
<img src="https://godoc.org/github.com/schollz/versionedtext?status.svg&style=flat-square" alt="Docs">
</a>

A simple library wrapping [sergi/go-diff](https://github.com/sergi/go-diff).

## Basic usage

```
d := versionedtext.NewVersionedText("The dog jumped over the fence.")
d.Update("The cat jumped over the fence.")
fmt.Println(d.GetPreviousByIndex(0))
// "The dog jumped over the fence."
fmt.Println(d.GetPreviousByIndex(1))
// "The cat jumped over the fence."
```

History can also be accessed by timestamps, more information in tests and in [the docs](https://godoc.org/github.com/schollz/versionedtext).


## Copyright and License

The original Google Diff, Match and Patch Library is licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0). The full terms of that license are included here in the [APACHE-LICENSE-2.0](/APACHE-LICENSE-2.0) file.

Diff, Match and Patch Library

> Written by Neil Fraser
> Copyright (c) 2006 Google Inc.
> <http://code.google.com/p/google-diff-match-patch/>

This Go version of Diff, Match and Patch Library is licensed under the [MIT License](http://www.opensource.org/licenses/MIT) (a.k.a. the Expat License) which is included here in the [LICENSE](/LICENSE) file.

Go version of Diff, Match and Patch Library

> Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
> <https://github.com/sergi/go-diff>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
