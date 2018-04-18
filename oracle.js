function findSimilarVectors(id, threshold) {
    var results = [];
    var v = storage.Find(id);
    
    if( v == null ) {
        return status.NotFound();
    }
        
    for( var cmp in storage.All() ) {
        if( v.Is(cmp) ){
            continue;
        }
        
        var dist_a = v.Jaccard( cmp, 0, 300);
        var dist_b = v.Cosine( cmp, 301, 500);
        if( ((dist_a + dist_b) / 2.0) <= threshold ) {
            results.append(cmp);
        }
    }

    return results;
}
