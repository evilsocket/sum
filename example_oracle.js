// compute the dot product of the vector 'id' with
// every other vectors in the storage
function dotAll(id) {
    var accu = 0.0;
    var v = records.Find(id);
    
    if( v.IsNull() == true ) {
        console.log("Vector " + id + " not found.");
        return null;
    }

    var all = records.All();
    var count = all.length;
    for( var i = 0; i < count; i++ ) {
        var cmp = all[i];
        if( v.Is(cmp) ){
            continue;
        }
        accu += v.Dot(cmp)
    }

    /* TODO: implement map/reduce logic:
    
        var accu = algo.Map( all, function(cmp){
            return v.Is(cmp) ? 0.0 : v.Dot(cmp);
        }).Reduce(function(a, b){
            return a + b;
        })

    */

    return accu;
}
