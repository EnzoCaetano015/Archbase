```java
package com.example.api.repository;

import com.example.api.domain.Item;
import org.springframework.data.mongodb.repository.Query;
import org.springframework.data.mongodb.repository.ReactiveMongoRepository;
import org.springframework.data.mongodb.repository.Update;
import org.springframework.stereotype.Repository;
import reactor.core.publisher.Mono;

@Repository
public interface ItemRepository extends ReactiveMongoRepository<Item, String> {

    @Update("{ '$set': { 'name': ?1 } }")
    @Query("{ '_id': ObjectId(?0) }")
    Mono<Void> updateNameById(String id, String name);

}
```
