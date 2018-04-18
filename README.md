<p align="center">
  <h1>SUM</h1>
  <p align="center">
    <a href="https://github.com/evilsocket/sum/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/evilsocket/sum.svg?style=flat-square"></a>
    <a href="https://github.com/evilsocket/sum/blob/master/LICENSE.md"><img alt="Software License" src="https://img.shields.io/badge/license-GPL3-brightgreen.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/evilsocket/sum"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/evilsocket/sum?style=flat-square"></a>
  </p>
</p>

Sum is a high performance service with one goal in mind: be a persistent and quick linear algebra server for large datasets.

**This is work in progress, do not use until v1.0.0 will be released.**

## Why

If you work with machine learning, deep learning, big data, distributed computing or any type of problem which requires
smart solutions for large datasets, you know you have **one problem** that we all have: storing your datasets.

Yes, you can serialize it in the most optimized and efficient format, have it as a csv file or in some exotic format that some framework requires, 
still it'll always be a **file**, and most likely a big one.

You have to load it (in memory, either ram or gpu memory), you have to perform stuff on it, you have to save it back ... **every single time**

Yes, you can use MySQL or even Redis to store it more or less efficiently, still the server is not very linear algebra oriented
and you have to offload most of the computation to your own machine (and still load it, edit it, save it, etc etc).

## What If ...

... there's a server exposing a clean and fast API to define, query and manipulate vectors in memory with data persistance on disk?

**This is what SUM is**.

## The Name

Sum is the third word of the ["cogito ergo sum"](https://en.wikipedia.org/wiki/Cogito_ergo_sum) Latin proposition, since **Cogito** is the name of one
of the systems I've been working for my current employer and it'll heavily rely on this server, I thought "sum" would be a cool name too :)

**[How to pronounce it](https://www.youtube.com/watch?v=ZnYgSukEk4A)**.

## License

`sum` is made with ♥  by [Simone 'evilsocket' Margaritelli](https://github.com/evilsocket) and it's released under the GPL 3 license.
