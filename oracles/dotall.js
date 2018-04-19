// compute the dot product of the vector 'id' with
// every other vectors in the storage
function dotAll(id) {
    var v = records.Find(id), n = 0.0;
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    records.AllBut(v).forEach(function(record){
        n += v.Dot(record);
    });

    return n;
}
