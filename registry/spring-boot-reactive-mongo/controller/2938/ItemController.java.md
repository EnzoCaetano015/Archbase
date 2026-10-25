```java
package com.example.api.controller;

import com.example.api.request.UpdateItemRequest;
import com.example.api.response.GetItemResponse;
import com.example.api.service.ItemService;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.NonNull;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

@RequestMapping(value = "item")
@Tag(name = "item")
@RequiredArgsConstructor
@RestController
public class ItemController {

    private final ItemService itemService;

    @Operation(summary = "Retrieves a item by its ID.", method = "GET")
    @GetMapping(value = "/{id}")
    public Mono<GetItemResponse> getItem(@PathVariable @NonNull String id) {
        return itemService.getItem(id);
    }

    @Operation(summary = "Creates a new item by passing its name.", method = "POST")
    @PostMapping
    public Mono<GetItemResponse> createItem(@RequestBody @NonNull String name) {
        return itemService.createItem(name);
    }

    @Operation(summary = "Update a item by passing its ID and name.", method = "PUT")
    @PutMapping
    public Mono<Void> updateItem(@RequestBody @NonNull UpdateItemRequest request) {
        return itemService.updateItem(request);
    }

    @Operation(summary = "Delete a item by passing its ID.", method = "DELETE")
    @DeleteMapping("/{id}")
    public Mono<Void> deleteItem(@PathVariable @NonNull String id) {
        return itemService.deleteItem(id);
    }

    @Operation(summary = "Get all registered items.", method = "GET")
    @GetMapping()
    public Flux<GetItemResponse> getItems() {
        return itemService.getItems();
    }
}
```
