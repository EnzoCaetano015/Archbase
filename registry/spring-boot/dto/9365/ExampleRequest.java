package com.example.api.domain.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record CreateItemRequest(
    @NotBlank
    @Size(max = 100)
    String name,

    @NotBlank
    @Size(max = 40)
    String code
) {}
