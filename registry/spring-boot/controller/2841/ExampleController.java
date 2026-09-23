package com.example.api.controller;

import com.example.api.domain.Item;
import com.example.api.domain.dto.CreateItemRequest;
import com.example.api.service.ItemService;
import io.swagger.v3.oas.annotations.Operation;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.*;

@RestController
@RequiredArgsConstructor
@RequestMapping("items")
public class ItemController {
    private final ItemService service;

    @GetMapping
    @Operation(summary = "List items")
    public Page<Item> list(Pageable pageable) {
        return service.list(pageable);
    }

    @PostMapping
    @Operation(summary = "Create an item")
    public Item create(@Valid @RequestBody CreateItemRequest request) {
        return service.create(request);
    }
}
