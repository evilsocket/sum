// compute the dot product of the vector 'id' with
// every other vectors in the storage
function dotAll(id) {
    var accu = 0.0;
    var v = records.Find(id);
    
    if( v.IsNull() == false ) {
        var all = records.All();
        var count = all.length;
        for( var i = 0; i < count; i++ ) {
            var cmp = all[i];
            if( v.Is(cmp) ){
                continue;
            }
            accu += v.Dot(cmp)
        }
    } else {
        console.log("Vector " + id + " not found.");
    }

    return accu;
}
