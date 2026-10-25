```java
package com.example.api.service;

import com.example.api.request.UpdateItemRequest;
import com.example.api.response.GetItemResponse;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

public interface ItemService {

    Mono<GetItemResponse> getItem(String id);

    Mono<GetItemResponse> createItem(String name);

    Mono<Void> updateItem(UpdateItemRequest request);

    Mono<Void> deleteItem(String id);

    Flux<GetItemResponse> getItems();

}
```
