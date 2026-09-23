package com.example.api.web;

import java.util.List;

public record ApiResponse<T>(boolean success, T data, List<String> errors) {
    public static <T> ApiResponse<T> success(T data) {
        return new ApiResponse<>(true, data, List.of());
    }

    public static ApiResponse<Void> error(List<String> errors) {
        return new ApiResponse<>(false, null, List.copyOf(errors));
    }
}
