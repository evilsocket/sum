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
