// Given the vector with id=`id`, return a list of
// other vectors which cosine similarity to the reference
// one is greater or equal than the threshold.
// Results are given as a dictionary of :
//      `vector_id => similarity`
function findSimilar(id, threshold) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("vector " + id + " not found.");
    }

    var results = {};
    var all = records.AllBut(v)
    var num = all.length;    
        
    for( var i = 0; i < num; ++i ) {
        var record = all[i];
        var similarity = v.Cosine(record);
        if( similarity >= threshold ) {
           results[record.ID] = similarity
        }
    }

    return results;
}
