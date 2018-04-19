// compute the dot product of the vector 'id' with
// every other vectors in the storage
function dotAll(id) {
    var accu = 0.0;
    var v = records.Find(id);
    
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var all = records.AllBut(v);
    var count = all.length;
    for( var i = 0; i < count; i++ ) {
        accu += v.Dot(all[i]);
    }

    /* TODO: implement map/reduce logic:
    
        var accu = ctx.Map( all, function(cmp){
            return v.Is(cmp) ? 0.0 : v.Dot(cmp);
        }).Reduce(function(a, b){
            return a + b;
        })
    */

    return accu;
}
