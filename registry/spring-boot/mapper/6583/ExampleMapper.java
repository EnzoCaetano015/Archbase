package com.example.api.mapper;

import com.example.api.domain.Item;
import com.example.api.domain.dto.CreateItemRequest;
import org.mapstruct.Mapper;
import org.mapstruct.Mapping;

@Mapper(componentModel = "spring")
public interface ItemMapper {
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "code", expression = "java(request.code().trim().toUpperCase())")
    Item toEntity(CreateItemRequest request);
}
