package com.example.api.service;

import com.example.api.domain.Item;
import com.example.api.domain.dto.CreateItemRequest;
import com.example.api.mapper.ItemMapper;
import com.example.api.repository.ItemRepository;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.web.server.ResponseStatusException;

@Service
@RequiredArgsConstructor
public class ItemService {
    private final ItemRepository repository;
    private final ItemMapper mapper;

    public Page<Item> list(Pageable pageable) {
        return repository.findAll(pageable);
    }

    @Transactional
    public Item create(CreateItemRequest request) {
        if (repository.existsByCode(request.code())) {
            throw new ResponseStatusException(HttpStatus.CONFLICT, "Code already exists");
        }
        return repository.save(mapper.toEntity(request));
    }
}
