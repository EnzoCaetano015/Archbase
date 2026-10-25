```java
package com.example.api.service.impl;

import com.example.api.domain.Item;
import com.example.api.repository.ItemRepository;
import com.example.api.request.UpdateItemRequest;
import com.example.api.response.GetItemResponse;
import com.example.api.service.ItemService;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import com.example.api.exception.ItemNotFoundException;

import java.util.function.Function;

@RequiredArgsConstructor
@Service
public class ItemServiceImpl implements ItemService {

    private final ItemRepository itemRepository;

    public Mono<GetItemResponse> getItem(String id) {
        Mono<Item> fallback = Mono.error(ItemNotFoundException::new);
        Function<Item, GetItemResponse> response = item -> new GetItemResponse(item.getId(), item.getName());

        return itemRepository.findById(id).switchIfEmpty(fallback).map(response);
    }

    public Mono<GetItemResponse> createItem(String name) {
        Item item = new Item(null, name);
        Function<Item, GetItemResponse> response = inserted -> new GetItemResponse(inserted.getId(), inserted.getName());

        return itemRepository.insert(item).map(response);
    }

    public Mono<Void> updateItem(UpdateItemRequest request) {
        return itemRepository.updateNameById(request.getId(), request.getName());
    }

    public Mono<Void> deleteItem(String id) {
        return itemRepository.deleteById(id);
    }

    public Flux<GetItemResponse> getItems() {
        Function<Item, GetItemResponse> response = items -> new GetItemResponse(items.getId(), items.getName());

        return itemRepository.findAll().map(response);
    }

}
```
