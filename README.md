<p align="center">
  <p align="center">
    <a href="https://github.com/evilsocket/sum/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/evilsocket/sum.svg?style=flat-square"></a>
    <a href="https://github.com/evilsocket/sum/blob/master/LICENSE.md"><img alt="Software License" src="https://img.shields.io/badge/license-GPL3-brightgreen.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/evilsocket/sum"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/evilsocket/sum?style=flat-square"></a>
  </p>
</p>

If you work with machine learning you probably find yourself having around a bunch of huge CSV files that maybe you 
keep using to train your models, or you run PCA on them, or you perform any sort of analysis. If this is the case, you 
know the struggle of:

* parsing and loading the file with `numpy`, `tensorflow` or whatever.
* crossing your fingers that your laptop can actually store those records in memory.
* running your algorithm
* ... waiting ...

This is an attempt to make these tedious tasks simpler if not completely automated. 

## What is SUM

Sum is a database and gRPC high performance service offering three main things:

1. Persistace for your vectors.
2. A simple CRUD system to create, read, update and delete them.
3. **Oracles**.

An **oracle** is a piece of javascript logic you want to run on your data, this code is sent to the Sum server by a 
client, compiled and stored. It'll then be available for every client to use in order to "query" the data.

For instance, this is the `findSimilar` oracle definition:

```js
// Given the vector with id=`id`, return a list of
// other vectors which cosine similarity to the reference
// one is greater or equal than the threshold.
// Results are given as a dictionary of :
//      `vector_id => similarity`
function findSimilar(id, threshold) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var results = {};
    records.AllBut(v).forEach(function(record){
        var similarity = v.Cosine(record);
        if( similarity >= threshold ) {
           results[record.Id] = similarity
        }
    });

    return results;
}
```

Once defined on the Sum server, any client will be able to execute calls like `findSimilar("some-vector-id-here", 0.9)`.

To have a better idea of how this works, take a look at [the example python code](https://github.com/evilsocket/sum/blob/master/example_client.py#L95) that will
create a few vectors on the server, define an oracle, call it for every vector and print the similarities the server returned.

**This is work in progress, do not use until v1.0.0 will be released.**

## License

`sum` is made with ♥  by [Simone 'evilsocket' Margaritelli](https://github.com/evilsocket) and it's released under the GPL 3 license.